import { dirname } from "path";
import { fileURLToPath } from "url";
import { FlatCompat } from "@eslint/eslintrc";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const compat = new FlatCompat({
  baseDirectory: __dirname,
});

const eslintConfig = [
  ...compat.extends("next/core-web-vitals", "next/typescript"),
  {
    ignores: [
      "node_modules/**",
      ".next/**",
      "out/**",
      "build/**",
      "next-env.d.ts",
    ],
  },
  {
    // Relax rules for legacy core files (will be refactored during plugin migration)
    files: ["src/core/**/*.ts", "src/core/**/*.tsx"],
    rules: {
      "@typescript-eslint/no-explicit-any": "warn", // Warn instead of error for legacy code
      "@typescript-eslint/no-empty-object-type": "warn", // Legacy domain models may extend empty interfaces
    },
  },
  {
    // Strict rules for new code (plugins, lib, components, app)
    files: [
      "src/plugins/**/*.ts",
      "src/plugins/**/*.tsx",
      "src/lib/**/*.ts",
      "src/lib/**/*.tsx",
      "src/components/**/*.ts",
      "src/components/**/*.tsx",
      "src/app/**/*.ts",
      "src/app/**/*.tsx",
      "src/features/**/*.ts",
      "src/features/**/*.tsx",
    ],
    rules: {
      "@typescript-eslint/no-explicit-any": "error", // Strict - no any allowed
    },
  },
];

export default eslintConfig;
