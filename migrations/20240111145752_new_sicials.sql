-- +goose Up
-- +goose StatementBegin
INSERT OR IGNORE INTO setting VALUES ('yLR1176FQj1BQks', 'social_facebook', '');
INSERT OR IGNORE INTO setting VALUES ('rKVq63So91kMuN7', 'social_instagram', '');
INSERT OR IGNORE INTO setting VALUES ('NVv27ea47Yo7gPm', 'social_twitter', '');
INSERT OR IGNORE INTO setting VALUES ('VjdMVG7LcUL274G', 'social_dribbble', '');
INSERT OR IGNORE INTO setting VALUES ('8sz9yVDNvNBa97b', 'social_github', '');
INSERT OR IGNORE INTO setting VALUES ('CoDDXfxF4GZxq6b', 'social_youtube', '');
INSERT OR IGNORE INTO setting VALUES ('AC3of7o9pS9HdB1', 'social_other', '');

-- Fix existing smtp_port values that might be '0'
UPDATE setting SET value = '' WHERE key = 'smtp_port' AND value = '0';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- NOTE: the previous Down removed rows by id, but two of those ids
-- (`CoDDXfxF4GZxq6b`, `AC3of7o9pS9HdB1`) are actually owned by the init
-- migration (mail_letter_purchase / smtp_host). The old Down was
-- destructive on any DB that reached the init migration. We now filter
-- by `key` so only the social rows this Up section is responsible for
-- are removed.
DELETE FROM setting WHERE key IN (
    'social_facebook',
    'social_instagram',
    'social_twitter',
    'social_dribbble',
    'social_github',
    'social_youtube',
    'social_other'
);
-- +goose StatementEnd