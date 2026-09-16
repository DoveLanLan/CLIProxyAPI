// Run with: node claude-vision-smoke.cjs <existing-settings.json> <card.png>
// Generate card.png from vision-card.svg with rsvg-convert first.
// Uses existing credentials in memory; never prints keys or image base64.
const fs = require('node:fs');
const path = require('node:path');
const { spawn } = require('node:child_process');

const [settingsPath, imagePath] = process.argv.slice(2);
if (!settingsPath || !imagePath) {
  console.error('Usage: node claude-vision-smoke.cjs <settings.json> <card.png>');
  process.exit(2);
}
const env = { ...process.env, ...JSON.parse(fs.readFileSync(settingsPath, 'utf8')).env };
delete env.CLAUDECODE;
if (!env.ANTHROPIC_API_KEY && env.ANTHROPIC_AUTH_TOKEN) {
  env.ANTHROPIC_API_KEY = env.ANTHROPIC_AUTH_TOKEN;
}
const file = path.resolve(imagePath);
const args = [
  '--bare', '--setting-sources', '', '--strict-mcp-config', '--disable-slash-commands',
  '--no-session-persistence', '--model', 'deepseek-v4.1-flash',
  '--tools', 'Read', '--allowedTools', 'Read', '--permission-mode', 'dontAsk',
  '--output-format', 'stream-json', '--verbose', '--max-budget-usd', '1', '-p',
  `Use the Read tool to open ${JSON.stringify(file)}. Report the exact text printed in the image and the colors and shapes from left to right. Do not infer from the filename.`,
];
const child = spawn('claude', args, { cwd: path.dirname(file), env, stdio: ['ignore', 'pipe', 'inherit'] });
let pending = '';
let sawImage = false;
let passed = false;
function handle(line) {
  if (!line.trim()) return;
  const event = JSON.parse(line);
  if (event.type === 'user') {
    sawImage ||= JSON.stringify(event.message).includes('"type":"image"');
    console.log(JSON.stringify({ type: 'tool_result', sawImage }));
  } else if (event.type === 'result') {
    const answer = event.result || '';
    passed = sawImage && !event.is_error && /VISION CHECK 7392/.test(answer)
      && /red.{0,30}square/is.test(answer) && /blue.{0,30}circle/is.test(answer)
      && /green.{0,30}triangle/is.test(answer);
    console.log(JSON.stringify({ session_id: event.session_id, is_error: event.is_error,
      num_turns: event.num_turns, result: answer, passed }));
  } else if (event.type === 'assistant') {
    console.log(JSON.stringify({ type: event.type, model: event.message?.model,
      content: event.message?.content }));
  }
}
child.stdout.on('data', chunk => {
  pending += chunk;
  const lines = pending.split('\n');
  pending = lines.pop();
  for (const line of lines) handle(line);
});
child.on('error', error => { console.error(error.message); process.exitCode = 1; });
child.on('exit', code => {
  handle(pending);
  process.exitCode = code === 0 && passed ? 0 : 1;
});
