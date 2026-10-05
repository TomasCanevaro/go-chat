ALTER TABLE conversations
ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT 'direct';

ALTER TABLE conversations
ADD CONSTRAINT valid_conversation_type
CHECK (type IN ('direct', 'group'));