package durable

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/canonical/sqlair"
	"github.com/google/uuid"
)

type Device struct {
	UUID               uuid.UUID `db:"uuid"`
	Name               string    `db:"name"`
	ChannelUUID        uuid.UUID `db:"channel_uuid"`
	ChannelName        *string   `db:"channel_name"`
	LastIP             *string   `db:"last_ip"`
	LastConnectTime    *string   `db:"last_connect_time"`
	LastDisconnectTime *string   `db:"last_disconnect_time"`
	DeviceInfo         *string   `db:"device_info"`
	DeviceInfoUpdated  *string   `db:"device_info_updated"`
}

func (store *Store) GetAllDevices(ctx context.Context) ([]Device, error) {
	resp := []Device{}
	err := store.View(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`SELECT (d.uuid, d.name, d.channel_uuid, c.name, d.last_ip, d.last_connect_time, d.last_disconnect_time, d.device_info, d.device_info_updated)
			     AS (&Device.uuid, &Device.name, &Device.channel_uuid, &Device.channel_name, &Device.last_ip, &Device.last_connect_time, &Device.last_disconnect_time, &Device.device_info, &Device.device_info_updated)
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
			`SELECT (d.uuid, d.name, d.channel_uuid, c.name, d.last_ip, d.last_connect_time, d.last_disconnect_time, d.device_info, d.device_info_updated)
			     AS (&Device.uuid, &Device.name, &Device.channel_uuid, &Device.channel_name, &Device.last_ip, &Device.last_connect_time, &Device.last_disconnect_time, &Device.device_info, &Device.device_info_updated)
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
				WHERE uuid = $Device.uuid`,
			Device{})
		err := tx.Query(stmt, device).Run()
		if err != nil {
			log.Printf("device modify failed: %v\n", err)
		}
		return err
	})
	return err
}

func (store *Store) LoginDevice(ctx context.Context, uuid uuid.UUID, remoteIP string) (*Device, error) {
	d := Device{}
	now := time.Now().UTC().Format(time.RFC3339)
	err := store.Update(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`SELECT (uuid, name, channel_uuid)
			     AS (&Device.uuid, &Device.name, &Device.channel_uuid)
			   FROM devices WHERE uuid = $M.uuid`,
			Device{},
			sqlair.M{})
		err := tx.Query(stmt, sqlair.M{"uuid": uuid}).Get(&d)
		if err == nil {
			// Device exists, update last_ip and last_connect_time
			stmt = sqlair.MustPrepare(
				`UPDATE devices SET last_ip = $M.ip, last_connect_time = $M.time WHERE uuid = $M.uuid`,
				sqlair.M{})
			err = tx.Query(stmt, sqlair.M{"uuid": uuid, "ip": remoteIP, "time": now}).Run()
			return err
		}
		if !errors.Is(err, sqlair.ErrNoRows) {
			return err
		}
		// New device, create with connection info
		d = Device{UUID: uuid, Name: uuid.String(), ChannelUUID: DefaultChannelUUID}
		stmt = sqlair.MustPrepare(
			`INSERT INTO devices (uuid, name, channel_uuid, last_ip, last_connect_time)
			 VALUES ($M.uuid, $M.name, $M.channel_uuid, $M.ip, $M.time)`,
			sqlair.M{})
		err = tx.Query(stmt, sqlair.M{
			"uuid":         uuid,
			"name":         d.Name,
			"channel_uuid": d.ChannelUUID,
			"ip":           remoteIP,
			"time":         now,
		}).Run()
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

func (store *Store) LogoutDevice(ctx context.Context, uuid uuid.UUID) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return store.Update(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`UPDATE devices SET last_disconnect_time = $M.time WHERE uuid = $M.uuid`,
			sqlair.M{})
		return tx.Query(stmt, sqlair.M{"uuid": uuid, "time": now}).Run()
	})
}

func (store *Store) DeleteDevice(ctx context.Context, uuid uuid.UUID) error {
	return store.Update(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`DELETE FROM devices WHERE uuid = $M.uuid`,
			sqlair.M{})
		return tx.Query(stmt, sqlair.M{"uuid": uuid}).Run()
	})
}

func (store *Store) UpdateDeviceInfo(ctx context.Context, deviceUUID uuid.UUID, info string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return store.Update(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`UPDATE devices SET device_info = $M.info, device_info_updated = $M.time WHERE uuid = $M.uuid`,
			sqlair.M{})
		return tx.Query(stmt, sqlair.M{"uuid": deviceUUID, "info": info, "time": now}).Run()
	})
}
