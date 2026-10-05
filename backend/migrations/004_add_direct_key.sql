ALTER TABLE conversations
ADD COLUMN direct_key VARCHAR(100);

CREATE UNIQUE INDEX unique_direct_conversation
ON conversations (direct_key)
WHERE type = 'direct';