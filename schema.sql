CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE quizzes (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    created_by INT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE questions (
    id SERIAL PRIMARY KEY,
    quiz_id INT REFERENCES quizzes(id),
    text TEXT NOT NULL,
    options TEXT[] NOT NULL,
    answer TEXT NOT NULL,
    image_url TEXT,
    context TEXT,
    word_definition JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE scores (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    quiz_id INT REFERENCES quizzes(id),
    score DECIMAL(5,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, quiz_id)
);

CREATE INDEX idx_scores_quiz_user ON scores(quiz_id, user_id);
CREATE INDEX idx_scores_score ON scores(score DESC); 