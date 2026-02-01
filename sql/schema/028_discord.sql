-- +goose Up
-- Update network constraint to include Discord
ALTER TABLE sources DROP CONSTRAINT network_check;

ALTER TABLE sources
ADD CONSTRAINT network_check CHECK (
    network IN (
        'Instagram',
        'Bluesky',
        'Murrtube',
        'BadPups',
        'TikTok',
        'Mastodon',
        'Reddit',
        'Telegram',
        'Discord',
        'YouTube',
        'FurTrack',
        'Google Analytics'
    )
);

-- +goose Down
ALTER TABLE sources DROP CONSTRAINT network_check;

ALTER TABLE sources
ADD CONSTRAINT network_check CHECK (
    network IN (
        'Instagram',
        'Bluesky',
        'Murrtube',
        'BadPups',
        'TikTok',
        'Mastodon',
        'Reddit',
        'Telegram',
        'YouTube',
        'FurTrack',
        'Google Analytics'
    )
);