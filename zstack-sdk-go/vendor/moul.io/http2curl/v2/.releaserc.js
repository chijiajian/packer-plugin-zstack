/**
 * Copyright (c) HashiCorp, Inc.
 */

module.exports = {
    branch: 'master',
    plugins: [
        '@semantic-release/commit-analyzer',
        '@semantic-release/release-notes-generator',
        '@semantic-release/github',
    ],
};
