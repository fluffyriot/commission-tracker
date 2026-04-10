-- +goose Up

CREATE TABLE tag_classifications (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    name VARCHAR(40) NOT NULL,
    CONSTRAINT fk_tag_classifications_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT unique_classification_name_per_user UNIQUE (user_id, name)
);

CREATE TABLE tags (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    classification_id UUID,
    name VARCHAR(40) NOT NULL,
    CONSTRAINT fk_tags_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_tags_classification FOREIGN KEY (classification_id) REFERENCES tag_classifications (id) ON DELETE SET NULL,
    CONSTRAINT unique_tag_name_per_user UNIQUE (user_id, name)
);

CREATE UNIQUE INDEX idx_unique_tag_name_per_classification
    ON tags (classification_id, name)
    WHERE classification_id IS NOT NULL;

CREATE TABLE post_tags (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    post_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    CONSTRAINT fk_post_tags_post FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE,
    CONSTRAINT fk_post_tags_tag FOREIGN KEY (tag_id) REFERENCES tags (id) ON DELETE CASCADE,
    CONSTRAINT unique_post_tag UNIQUE (post_id, tag_id)
);

CREATE INDEX idx_tag_classifications_user_id ON tag_classifications (user_id);
CREATE INDEX idx_tags_user_id ON tags (user_id);
CREATE INDEX idx_tags_classification_id ON tags (classification_id);
CREATE INDEX idx_post_tags_post_id ON post_tags (post_id);
CREATE INDEX idx_post_tags_tag_id ON post_tags (tag_id);

-- +goose Down

DROP TABLE post_tags;
DROP TABLE tags;
DROP TABLE tag_classifications;