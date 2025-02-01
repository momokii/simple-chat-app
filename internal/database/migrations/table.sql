CREATE TYPE gender_enum AS ENUM ('male', 'female');
CREATE TYPE range_age_enum AS ENUM ('18-24', '25-30', '31-40', '41-50');
CREATE TYPE language_enum AS ENUM ('indonesia', 'english');
-- for edit enum data
-- ALTER TYPE gender_enum ADD VALUE 'other';

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(25) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE room_chat (
    id SERIAL PRIMARY KEY,
    code VARCHAR(25) NOT NULL UNIQUE,
    created_by INT NOT NULL REFERENCES users(id),
    name VARCHAR(25) NOT NULL,
    description VARCHAR(255) NOT NULL,
    password VARCHAR(255) DEFAULT '',
    is_private BOOLEAN DEFAULT FALSE,
    is_train_room BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE room_chat_train (
    id SERIAL PRIMARY KEY,
    room_code VARCHAR(25) NOT NULL REFERENCES room_chat(code) ON DELETE CASCADE,
    gender gender_enum NOT NULL,
    language language_enum NOT NULL,
    range_age range_age_enum NOT NULL,
    employment_type TEXT NOT NULL,
    description TEXT NOT NULL,
    hobby TEXT NOT NULL,
    personality TEXT NOT NULL,
    is_still_continue BOOLEAN DEFAULT TRUE,
    UNIQUE (room_code)
);

CREATE TABLE messages (
    id SERIAL PRIMARY KEY,
    room_id INT NOT NULL REFERENCES room_chat(id) ON DELETE CASCADE,
    sender_id INT NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE room_members (
    id SERIAL PRIMARY KEY,
    room_id INT NOT NULL REFERENCES room_chat(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (room_id, user_id)
);

-- index for table users
CREATE INDEX idx_messages_room_id ON messages(room_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);

-- create assistant base user for assistant user data for messaging training with id 0
INSERT INTO users (id, username, password) VALUES (0, 'assistant', '$2y$10$$2a$16$w9H/xLUqZ0RDgUe0PHsQZuT2.BOvkTqWEcXLW.EqHNliDjqSbHKHa');