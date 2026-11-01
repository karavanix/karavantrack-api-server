-- The language of the user's emails and push notifications: the one the web
-- cabinet or the mobile app is switched to. Users from before it was
-- recorded get Russian, which every text was written in until then.
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS language varchar(8) NOT NULL DEFAULT 'ru';

ALTER TABLE users
  ADD CONSTRAINT users_language_check CHECK (language IN ('ru', 'uz', 'en'));
