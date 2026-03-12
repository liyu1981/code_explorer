ALTER TABLE tasks ADD COLUMN initiator_id TEXT REFERENCES tasks(id);
CREATE INDEX idx_tasks_initiator_id ON tasks(initiator_id);
