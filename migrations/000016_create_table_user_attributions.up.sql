-- Where a user came from when they signed up: the landing page's UTM tags
-- (and click ids) carried into the web cabinet, or 'direct' when there were
-- none. Written once, when the user is created, and never overwritten.
CREATE TABLE IF NOT EXISTS user_attributions (
  user_id       uuid,
  source        varchar(128) NOT NULL,
  medium        varchar(128),
  campaign      varchar(128),
  content       varchar(128),
  term          varchar(128),
  gclid         varchar(256),
  yclid         varchar(256),
  referrer      varchar(255),
  landing_lang  varchar(8),
  created_at    timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id),
  CONSTRAINT user_attributions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS user_attributions_source_medium_campaign_idx
  ON user_attributions (source, medium, campaign);

-- Users created before attribution was recorded count as direct.
INSERT INTO user_attributions (user_id, source, created_at)
SELECT id, 'direct', COALESCE(created_at, now()) FROM users
ON CONFLICT (user_id) DO NOTHING;
