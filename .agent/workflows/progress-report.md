---
description: Generate a progress report after every user prompt
---

After completing work for each user prompt, generate a concise progress report as a markdown artifact at:
`/Users/fabiano/.gemini/antigravity/brain/cf877a10-fea1-41b4-8b4a-bd6f72d3332d/progress_report.md`

The report should include:
1. **Session timestamp** — current date/time
2. **What was done** — bullet list of changes made this prompt
3. **Files changed** — list of files created/modified/deleted
4. **Test status** — last known test results (pass/fail count)
5. **Build status** — compilation status
6. **Overall progress** — percentage or visual bar showing MVP completion
7. **Next steps** — what remains to be done

Overwrite the file each time to keep it current. Keep it concise (under 60 lines).
