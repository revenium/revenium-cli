## [1.4.0] - 2026-08-21

### Added

- **Skill tracking.** `meter completion` accepts `--skill-name`, `--skill-invocation-trigger`, `--skill-source`, `--skill-kind`, `--skill-plugin-name`, and `--skill-marketplace-name`, sent only when passed. These fields are specific to the completion endpoint — the audio, image, video, and tool-event schemas do not define them.

