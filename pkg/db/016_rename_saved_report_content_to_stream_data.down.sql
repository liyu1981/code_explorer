-- Rename stream_data column back to content in saved_reports
ALTER TABLE saved_reports RENAME COLUMN stream_data TO content;
