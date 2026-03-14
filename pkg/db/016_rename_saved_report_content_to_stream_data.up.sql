-- Rename content column to stream_data in saved_reports
ALTER TABLE saved_reports RENAME COLUMN content TO stream_data;
