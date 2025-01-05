-- Create Users Table
CREATE TABLE users (
    id INT PRIMARY KEY, -- Unique user identifier
    name VARCHAR(50) NOT NULL, -- Full name
    nickname VARCHAR(20) UNIQUE, -- Unique username
    email VARCHAR(60) UNIQUE NOT NULL, -- Unique email address
    status_on BOOLEAN DEFAULT TRUE -- Online status (default is true)
);

-- Create Chats Table
CREATE TABLE chats (
    id INT PRIMARY KEY -- Unique chat identifier
);

-- Create Messages Table
CREATE TABLE messages (
    id INT PRIMARY KEY, -- Unique message identifier
    id_c INT NOT NULL, -- Chat ID (links to a chat)
    id_u INT NOT NULL, -- User ID (sender of the message)
    message TEXT NOT NULL, -- Message content
    timestamp DATETIME NOT NULL -- Message send/scheduled time
);

-- Create Chats_Users Table
CREATE TABLE chats_users (
    id_c INT NOT NULL, -- Chat ID
    id_u INT NOT NULL, -- User ID
    PRIMARY KEY (id_c, id_u) -- Composite primary key ensures no duplicate entries
);

-- Create Groups Table
CREATE TABLE groups (
    id INT PRIMARY KEY, -- Unique group identifier
    name VARCHAR(50) NOT NULL -- Group name
);

-- Create Groups_Users Table
CREATE TABLE groups_users (
    id_g INT NOT NULL, -- Group ID
    id_u INT NOT NULL, -- User ID
    PRIMARY KEY (id_g, id_u) -- Composite primary key ensures no duplicate entries
);
