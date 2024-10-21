CREATE TABLE IF NOT EXISTS socket_data (
    id uuid PRIMARY KEY, -- unique id
    data BYTEA NOT NULL  --data from client

);
