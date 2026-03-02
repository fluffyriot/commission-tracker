-- +goose Up
ALTER TABLE sources DROP CONSTRAINT network_check;

ALTER TABLE sources
ADD CONSTRAINT network_check CHECK (
    network IN (
        'Instagram',
        'Twitter',
        'Threads',
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
        'DeviantArt',
        'e621',
        'Weasyl',
        'FurTrack',
        'FurAffinity',
        'Google Analytics',
        'Google Search Console'
    )
);

-- +goose Down
ALTER TABLE sources DROP CONSTRAINT network_check;

ALTER TABLE sources
ADD CONSTRAINT network_check CHECK (
    network IN (
        'Instagram',
        'Threads',
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
        'DeviantArt',
        'e621',
        'Weasyl',
        'FurTrack',
        'FurAffinity',
        'Google Analytics',
        'Google Search Console'
    )
);
