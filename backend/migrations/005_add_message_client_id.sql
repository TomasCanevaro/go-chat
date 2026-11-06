-- NULL preserves existing messages and allows sends without a correlation ID.
ALTER TABLE messages ADD COLUMN client_id VARCHAR(128);

ALTER TABLE messages
ADD CONSTRAINT messages_sender_client_id_key UNIQUE (sender_id, client_id);
