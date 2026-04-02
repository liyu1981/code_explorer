-- down migration for sqlitefs
DROP TABLE IF EXISTS fs_file_chunks;
DROP TABLE IF EXISTS fs_nodes;