module.exports = {
    packagerConfig: {
        name: 'EntropyTunnel',
        executableName: 'entropy-tunnel',
        icon: './icon',
        asar: true,
        extraResource: ['../bin'],
    },
    makers: [
        {
            name: '@electron-forge/maker-zip',
            platforms: ['darwin', 'linux', 'win32'],
        },
        {
            name: '@electron-forge/maker-dmg',
            config: {
                name: 'EntropyTunnel',
                format: 'ULFO',
            },
        },
        {
            name: '@electron-forge/maker-squirrel',
            config: {
                name: 'EntropyTunnel',
            },
        },
        {
            name: '@electron-forge/maker-deb',
            config: {
                options: {
                    maintainer: 'EntropyTunnel',
                    homepage: 'https://github.com/fabiano/entropy-tunnel',
                },
            },
        },
    ],
};
