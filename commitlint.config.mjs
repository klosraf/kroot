/**
 * Commit message linting.
 *
 * The type allow-list mirrors the Conventional Commit prefixes documented in
 * docs/enterprise/branching-strategy.md. Severity 1 = warn, 2 = error.
 */

export default {
  extends: ["@commitlint/config-conventional"],
  rules: {
    // The set matches the Conventional Commits specification, which is also
    // what the repository's global commit-msg hook accepts, so a message that
    // passes locally is never rejected by CI over its type alone.
    "type-enum": [
      2,
      "always",
      [
        "feat",
        "fix",
        "docs",
        "style",
        "refactor",
        "perf",
        "test",
        "build",
        "ci",
        "chore",
        "revert",
      ],
    ],
    // A scope identifies the affected area, e.g. `feat(server): ...`.
    // Warned rather than enforced so genuinely scope-less chores still land.
    "scope-empty": [1, "never"],
    // Acronyms (CLI, HTTP, API, SQL) are legitimate inside a subject and are
    // required in upper case by docs/enterprise/coding-standards.md. The rule
    // therefore forbids only wholesale capitalisation of the subject, while
    // allowing lower-case text and mid-sentence acronyms.
    "subject-case": [
      2,
      "never",
      ["sentence-case", "start-case", "pascal-case", "upper-case"],
    ],
    "subject-empty": [2, "never"],
    "subject-full-stop": [2, "never", "."],
    "header-max-length": [2, "always", 72],
    "body-max-line-length": [1, "always", 100],
  },
};
