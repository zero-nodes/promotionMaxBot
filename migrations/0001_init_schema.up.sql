CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    id_max BIGINT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    status INTEGER NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now()
);

CREATE TABLE tiket (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    id_max BIGINT NOT NULL,
    photo BYTEA,
    status INTEGER NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),

    CONSTRAINT fk_tiket_user
        FOREIGN KEY (id_max)
        REFERENCES users (id_max)
        ON DELETE CASCADE
);

CREATE INDEX idx_users_id_telegram ON users(id_max);
CREATE INDEX idx_tiket_user_date ON tiket(id_max, date);
CREATE INDEX idx_tiket_status ON tiket(status);
