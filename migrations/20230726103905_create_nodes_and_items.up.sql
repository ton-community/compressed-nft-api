CREATE TABLE items (
    id bigint NOT NULL PRIMARY KEY,
    owner character(48) NOT NULL
);

CREATE TABLE nodes (
    index bigint NOT NULL,
    version integer NOT NULL,
    hash bytea NOT NULL,
    PRIMARY KEY (index, version)
);