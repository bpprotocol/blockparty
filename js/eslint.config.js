import tseslint from "typescript-eslint";

// Non-type-aware flat config: fast, and independent of any single TypeScript
// version's type-checker. Type safety is enforced separately by `tsc`.
export default tseslint.config(
  { ignores: ["**/dist/**", "**/node_modules/**"] },
  ...tseslint.configs.recommended,
);
