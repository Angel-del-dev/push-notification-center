-- FIXME this is PSEUDO CODE, correct later

CREATE TABLE push_subscriptions (
  id SERIAL PRIMARY KEY,
  endpoint TEXT NOT NULL unique,
  p256dh TEXT NOT NULL,
  auth TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);