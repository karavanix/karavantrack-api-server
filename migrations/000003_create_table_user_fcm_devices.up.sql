CREATE TABLE IF NOT EXISTS user_fcm_devices (
  id            bigserial,
  user_id       uuid NOT NULL,
  device_id     text NOT NULL,
  device_name   varchar(255),
  device_type   varchar(255),
  device_token  text NOT NULL,
  expires_at    timestamptz,
  created_at    timestamptz DEFAULT now(),
  updated_at    timestamptz,
  PRIMARY KEY (id),
  CONSTRAINT user_fcm_devices_user_id_device_id_key UNIQUE (user_id, device_id),
  CONSTRAINT user_fcm_devices_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);
