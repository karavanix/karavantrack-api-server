CREATE TABLE IF NOT EXISTS user_fcm_devices(
    id bigserial,
    user_id uuid NOT NULL,
    device_id text NOT NULL,
    device_name varchar(255),
    device_type varchar(255),
    device_token text NOT NULL,
    expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone,
    PRIMARY KEY (id),
    UNIQUE (user_id, device_id),
    CONSTRAINT user_fcm_devices_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS user_fcm_devices_user_id_idx ON user_fcm_devices(user_id);

CREATE INDEX IF NOT EXISTS user_fcm_devices_device_id_idx ON user_fcm_devices(device_id);

CREATE INDEX IF NOT EXISTS user_fcm_devices_expires_at_idx ON user_fcm_devices(expires_at);

