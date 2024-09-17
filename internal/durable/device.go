package durable

import (
	"context"
	"errors"
	"log"

	"github.com/canonical/sqlair"
	"github.com/google/uuid"
)

type Device struct {
	UUID        uuid.UUID `db:"uuid"`
	Name        string    `db:"name"`
	ChannelUUID uuid.UUID `db:"channel_uuid"`
	ChannelName *string   `db:"channel_name"`
}

func (store *Store) GetAllDevices(ctx context.Context) ([]Device, error) {
	resp := []Device{}
	err := store.View(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`SELECT (d.uuid, d.name, d.channel_uuid, c.name)
			     AS (&Device.uuid, &Device.name, &Device.channel_uuid, &Device.channel_name)
			   FROM devices d
		       LEFT JOIN channels c ON d.channel_uuid = c.uuid COLLATE NOCASE`,
			Device{})
		err := tx.Query(stmt).GetAll(&resp)
		return err
	})
	return resp, err
}

func (store *Store) GetDeviceByUUID(ctx context.Context, uuid uuid.UUID) (*Device, error) {
	resp := Device{}
	err := store.View(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`SELECT (d.uuid, d.name, d.channel_uuid, c.name)
			     AS (&Device.uuid, &Device.name, &Device.channel_uuid, &Device.channel_name)
			   FROM devices d
		       LEFT JOIN channels c ON d.channel_uuid = c.uuid COLLATE NOCASE
			   WHERE d.uuid = $M.uuid`,
			Device{},
			sqlair.M{})
		err := tx.Query(stmt, sqlair.M{"uuid": uuid}).Get(&resp)
		if err != nil {
			log.Printf("failed to get device: %v\n", err)
		}
		return err
	})
	return &resp, err
}

func (store *Store) ModifyDevice(ctx context.Context, device *Device) error {
	err := store.Update(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`UPDATE devices
			      SET name = $Device.name,
				      channel_uuid = $Device.channel_uuid
				WHERE id = $Device.id`,
			Device{})
		err := tx.Query(stmt, device).Run()
		if err != nil {
			log.Printf("device modify failed: %v\n", err)
		}
		return err
	})
	return err
}

func (store *Store) LoginDevice(ctx context.Context, uuid uuid.UUID) (*Device, error) {
	d := Device{}
	err := store.Update(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`SELECT (uuid, name, channel_uuid)
			     AS (&Device.*)
			   FROM devices WHERE uuid = $M.uuid`,
			Device{},
			sqlair.M{})
		err := tx.Query(stmt, sqlair.M{"uuid": uuid}).Get(&d)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sqlair.ErrNoRows) {
			return err
		}
		d = Device{UUID: uuid, Name: uuid.String(), ChannelUUID: DefaultChannelUUID}
		stmt = sqlair.MustPrepare(
			`INSERT INTO devices (uuid, name, channel_uuid) VALUES ($Device.*)`,
			Device{})
		err = tx.Query(stmt, d).Run()
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &d, nil
}
