package durable

import (
	"context"
	ne "errors"
	"log"

	"github.com/canonical/sqlair"
	"github.com/google/uuid"

	"github.com/joe714/pixelgw/internal/errors"
)

var DefaultChannelUUID = uuid.MustParse("76ffcb18-d3c7-40d5-abea-3fe86d02a4ba")

type ChannelApplet struct {
	UUID   uuid.UUID `db:"uuid"`
	Idx    int       `db:"idx"`
	AppID  string    `db:"app_id"`
	Config *string   `db:"config"`
}

type ChannelSubscriber struct {
	UUID uuid.UUID `db:"uuid"`
	Name string    `db:"name"`
}

type Channel struct {
	UUID        uuid.UUID `db:"uuid"`
	Name        string    `db:"name"`
	Comment     *string   `db:"comment"`
	Applets     []ChannelApplet
	Subscribers []ChannelSubscriber
}

func (store *Store) CreateChannel(ctx context.Context, name string, comment *string) (*Channel, error) {
	ch := Channel{}
	stmt := sqlair.MustPrepare("SELECT &Channel.* FROM channels WHERE name = $M.name", Channel{}, sqlair.M{})
	err := store.DB.Query(ctx, stmt, sqlair.M{"name": name}).Get(&ch)
	if err == nil {
		return nil, errors.Wrap(errors.ChannelExists,
			"Channel %v already exists with uuid %v",
			ch.Name,
			ch.UUID)
	} else if !ne.Is(err, sqlair.ErrNoRows) {
		return nil, err
	}

	uuid, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	ch = Channel{
		UUID:    uuid,
		Name:    name,
		Comment: comment,
	}
	stmt = sqlair.MustPrepare("INSERT INTO channels (*) VALUES($Channel.*)", Channel{})
	err = store.DB.Query(ctx, stmt, &ch).Run()
	if err != nil {
		log.Printf("Error creating channel: %v\n", err)
		return nil, err
	}
	return &ch, nil
}

func (store *Store) GetAllChannels(ctx context.Context) ([]Channel, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	stmt := sqlair.MustPrepare("SELECT &Channel.* FROM channels ORDER BY name", Channel{})
	var res []Channel
	err := store.DB.Query(ctx, stmt).GetAll(&res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (store *Store) GetChannelByUUID(ctx context.Context, uuid uuid.UUID) (*Channel, error) {
	var ch Channel
	m := sqlair.M{"uuid": uuid}
	err := store.View(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare("SELECT &Channel.* FROM channels WHERE uuid = $M.uuid", Channel{}, sqlair.M{})
		err := tx.Query(stmt, m).Get(&ch)
		if err != nil {
			return err
		}

		stmt = sqlair.MustPrepare(
			"SELECT &ChannelApplet.* FROM channel_applets WHERE channel_uuid = $M.uuid ORDER BY idx",
			ChannelApplet{},
			sqlair.M{})
		err = tx.Query(stmt, m).GetAll(&ch.Applets)
		if err != nil && !ne.Is(err, sqlair.ErrNoRows) {
			return err
		}

		stmt = sqlair.MustPrepare(
			"SELECT &ChannelSubscriber.* FROM devices WHERE channel_uuid = $M.uuid ORDER BY name",
			ChannelSubscriber{},
			sqlair.M{})
		err = tx.Query(stmt, m).GetAll(&ch.Subscribers)
		if err != nil && !ne.Is(err, sqlair.ErrNoRows) {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (store *Store) GetChannelByName(ctx context.Context, name string) (*Channel, error) {
	var ch Channel
	m := sqlair.M{"name": name}
	err := store.View(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`SELECT &Channel.* FROM channels WHERE name = $M.name`,
			Channel{},
			sqlair.M{})
		err := tx.Query(stmt, m).Get(&ch)
		if ne.Is(err, sqlair.ErrNoRows) {
			return errors.ChannelNotFound
		} else {
			return err
		}
		return nil
	})
	return &ch, err
}

func (store *Store) CreateChannelApplet(ctx context.Context, channelUUID uuid.UUID, app *ChannelApplet) error {
	if uuid.Nil == app.UUID {
		uuid, err := uuid.NewV7()
		if err != nil {
			return err
		}
		app.UUID = uuid
	}
	err := store.Update(ctx, func(tx *TX) error {
		m := sqlair.M{"channel_uuid": channelUUID}
		stmt := sqlair.MustPrepare(
			`SELECT &Channel.* FROM channels WHERE uuid = $M.channel_uuid`,
			Channel{},
			sqlair.M{})
		ch := Channel{}
		err := tx.Query(stmt, m).Get(&ch)
		if err != nil {
			return err
		}

		count, err := appletCount(tx, channelUUID)
		if err != nil {
			return err
		}

		if app.Idx == -1 {
			app.Idx = count
		}
		if app.Idx > count {
			return errors.AppIndexOutOfRange
		}
		if app.Idx < count {
			err = reorder(tx, channelUUID, count, app.Idx)
			if err != nil {
				return err
			}
		}

		stmt = sqlair.MustPrepare(
			`INSERT INTO channel_applets (*) 
				VALUES ($M.channel_uuid, $ChannelApplet.*)`,
			sqlair.M{},
			ChannelApplet{})
		err = tx.Query(stmt, m, app).Run()
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func (store *Store) DeleteChannelApplet(ctx context.Context, channelUUID uuid.UUID, appletUUID uuid.UUID) error {
	log.Printf("Delete applet %v (channel %v)\n", appletUUID, channelUUID)
	err := store.Update(ctx, func(tx *TX) error {

		m := sqlair.M{"channel_uuid": channelUUID, "uuid": appletUUID}
		stmt := sqlair.MustPrepare(
			`SELECT &ChannelApplet.* FROM channel_applets
		         WHERE uuid = $M.uuid
				   AND channel_uuid = $M.channel_uuid`,
			ChannelApplet{},
			sqlair.M{})
		app := ChannelApplet{}
		err := tx.Query(stmt, m).Get(&app)
		if err != nil {
			log.Printf("Failed to get applet: %v", err)
			return err
		}

		stmt = sqlair.MustPrepare(
			`DELETE FROM channel_applets WHERE uuid = $M.uuid`,
			sqlair.M{})
		err = tx.Query(stmt, m).Run()
		if err != nil {
			log.Printf("Delete failed: %v\n", err)
			return err
		}

		count, err := appletCount(tx, channelUUID)
		if err != nil {
			return err
		}
		if app.Idx < count {
			err = reorder(tx, channelUUID, app.Idx, count)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (store *Store) ModifyChannelApplet(ctx context.Context, channelUUID uuid.UUID, appletUUID uuid.UUID, idx *int, cfg *string) error {

	err := store.Update(ctx, func(tx *TX) error {
		app := ChannelApplet{}
		stmt := sqlair.MustPrepare(
			`SELECT &ChannelApplet.* FROM channel_applets
		    	WHERE uuid = $M.uuid AND channel_uuid = $M.channel_uuid`,
			ChannelApplet{},
			sqlair.M{})
		m := sqlair.M{"channel_uuid": channelUUID, "uuid": appletUUID}
		err := tx.Query(stmt, m).Get(&app)
		if err != nil {
			return err
		}

		if cfg != nil {
			app.Config = cfg
			stmt = sqlair.MustPrepare(
				`UPDATE channel_applets SET config = $ChannelApplet.config
				    WHERE uuid = $ChannelApplet.uuid`,
				ChannelApplet{})
			err = tx.Query(stmt, app).Run()
			if err != nil {
				log.Printf("Failed updating applet config: %v\n", err)
				return err
			}
		}

		if idx != nil && *idx != app.Idx {
			log.Printf("Change applet %v original idx: %d new idx: %d\n", app.UUID, app.Idx, *idx)
			count, err := appletCount(tx, channelUUID)
			if err != nil {
				return err
			}
			if *idx > count {
				return errors.AppIndexOutOfRange
			}

			err = reorder(tx, channelUUID, app.Idx, *idx)
			if err != nil {
				log.Printf("Error reordering: %v\n", err)
				return err
			}
		}
		return nil
	})
	return err
}

func appletCount(tx *TX, channelUUID uuid.UUID) (int, error) {
	count := Count{}
	stmt := sqlair.MustPrepare(
		`SELECT count(*) AS &Count.count FROM channel_applets
		    WHERE channel_uuid = $M.channel_uuid`,
		Count{},
		sqlair.M{})
	err := tx.Query(stmt, sqlair.M{"channel_uuid": channelUUID}).Get(&count)
	if err != nil {
		log.Printf("Failed to get channel applet count: %v", err)
	}
	return count.Count, err
}

func reorder(tx *TX, channelUUID uuid.UUID, curIdx int, newIdx int) error {
	log.Printf("Reorder %v %d -> %d\n", channelUUID, curIdx, newIdx)
	if curIdx == newIdx {
		return nil
	}
	stmt := sqlair.MustPrepare(
		`UPDATE channel_applets SET idx = $M.newIdx
		    WHERE channel_uuid = $M.channel_uuid
		  	  AND idx = $M.idx`,
		sqlair.M{})
	m := sqlair.M{"channel_uuid": channelUUID, "idx": curIdx, "newIdx": -(newIdx + 100)}
	err := tx.Query(stmt, m).Run()
	if err != nil {
		log.Printf("Error moving applet %v: %v\n", m, err)
		return err
	}

	m = sqlair.M{"channel_uuid": channelUUID}
	if curIdx < newIdx {
		m["offset"] = 99
		m["lo"] = curIdx + 1
		m["hi"] = newIdx
	} else {
		m["offset"] = 101
		m["lo"] = newIdx
		m["hi"] = curIdx - 1
	}
	stmt = sqlair.MustPrepare(
		`UPDATE channel_applets SET idx = -(idx + $M.offset)
				    WHERE channel_uuid = $M.channel_uuid
					  AND idx >= $M.lo
					  AND idx <= $M.hi`,
		sqlair.M{})
	err = tx.Query(stmt, m).Run()
	if err != nil {
		log.Printf("Error shifting %v: %v\n", m, err)
		return err
	}
	stmt = sqlair.MustPrepare(
		`UPDATE channel_applets SET idx = -idx - 100
			        WHERE channel_uuid = $M.channel_uuid
				      AND idx < 0`,
		sqlair.M{})
	err = tx.Query(stmt, m).Run()
	if err != nil {
		log.Printf("Error restoring: %v\n", err)
		return err
	}
	return nil
}
