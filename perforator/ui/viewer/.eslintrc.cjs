module.exports = {
    root: true,
    env: {browser: true, es2020: true},
    extends: [
        'eslint:recommended',
        'plugin:react-hooks/recommended',
        'plugin:@typescript-eslint/base'
    ],
    plugins: ["@typescript-eslint", 'react-refresh'],
    ignorePatterns: ['dist', '.eslintrc.cjs'],
    parser: '@typescript-eslint/parser',
    rules: {
        'react-refresh/only-export-components': [
            'error',
            {allowConstantExport: true},
        ],
        "no-unused-vars": "off",
        "@typescript-eslint/no-unused-vars": ["error"]
    },
    root: true
}
