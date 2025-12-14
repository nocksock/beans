---
# beans-fgn2
title: Extend beans prime to include milestone context
status: scrapped
type: feature
priority: normal
created_at: 2025-12-14T11:37:59Z
updated_at: 2025-12-14T17:36:08Z
parent: beans-f11p
---

## Summary

The `beans prime` command generates a prompt that helps agents understand the beans system. It should be extended to inject additional project-specific knowledge, specifically:

1. **List of milestones** - Show all milestones defined in the project
2. **Current milestone(s)** - Highlight which milestone(s) are currently in-progress
3. **Milestone progress** - Optionally show completion status (e.g., X of Y children completed)

## Motivation

When an agent starts working on a project, understanding the current milestone context helps them:
- Prioritize work that aligns with current goals
- Understand the broader project roadmap
- Make better decisions about which beans to pick up

## Checklist

- [x] Query for all milestones in the project
- [x] Identify in-progress milestones
- [x] Format milestone information in a clear, readable way for the prompt
- [ ] Include milestone progress (completed/total children) if available (not requested)
- [x] Add the milestone section to the prime output
- [x] Test with projects that have no milestones (graceful handling)
- [x] Test with projects that have multiple in-progress milestones