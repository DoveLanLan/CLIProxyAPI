# Requirements

- Chat tool messages must not carry image_url content parts.
- Retain every tool response ID and text; retain exact image URLs and metadata.
- Emit images in a user message after all adjacent tool messages, never between
  parallel tool responses. Associate each image group with its tool-call ID.
- Do not change ordinary user images or text-only tool requests.
- Apply equally to streaming and non-streaming OpenAI-compatible Chat requests;
  do not apply to Responses compact endpoints.
- Validate with strict upstream mocks, full tests/build, and actual Claude Code
  Read of a synthetic PNG. Deploy through the existing git/workflow path and
  verify production version and successful image recognition.
