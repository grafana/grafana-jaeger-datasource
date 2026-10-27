process.env.TZ = 'Pacific/Easter';

const baseConfig = require('./.config/jest.config');
const { grafanaESModules, nodeModulesToTransform } = require('./.config/jest/utils');

module.exports = {
  ...baseConfig,
  // ESM-only packages pulled in by @grafana/* 13.3
  transformIgnorePatterns: [
    nodeModulesToTransform([...grafanaESModules, '@react-hookz/web', '@ver0/deep-equal', '@marcbachmann/cel-js']),
  ],
  transform: {
    '^.+\\.(t|j)sx?$': [
      '@swc/jest',
      {
        sourceMaps: 'inline',
        jsc: {
          parser: {
            syntax: 'typescript',
            tsx: true,
            decorators: false,
            dynamicImport: true,
          },
          transform: {
            react: {
              runtime: 'automatic',
            },
          },
        },
      },
    ],
  },
};
