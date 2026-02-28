-- +goose Up
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
        'Twitch',
        'YouTube',
        'e621',
        'FurTrack',
        'FurAffinity',
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
        'Discord',
        'YouTube',
        'e621',
        'FurTrack',
        'FurAffinity',
        'Google Analytics'
    )
);
