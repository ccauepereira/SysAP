import { spawn } from "node:child_process";

const defaultCaptureLimit = 1024 * 1024;

export function runCommand(command, args, options = {}) {
  return run(command, args, { ...options, capture: false });
}

export function runCommandCapture(command, args, options = {}) {
  return run(command, args, { ...options, capture: true });
}

function run(command, args, options) {
  const capture = options.capture;
  const outputLimit = options.outputLimit ?? defaultCaptureLimit;

  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      cwd: options.cwd,
      env: options.env,
      shell: false,
      stdio: capture ? ["ignore", "pipe", "pipe"] : "inherit",
    });
    const stdout = [];
    const stderr = [];
    let stdoutBytes = 0;
    let stderrBytes = 0;
    let outputExceeded = false;

    if (capture) {
      child.stdout.on("data", (chunk) => {
        stdoutBytes += chunk.length;
        if (stdoutBytes <= outputLimit) {
          stdout.push(chunk);
        } else {
          outputExceeded = true;
        }
      });
      child.stderr.on("data", (chunk) => {
        stderrBytes += chunk.length;
        if (stderrBytes <= outputLimit) {
          stderr.push(chunk);
        } else {
          outputExceeded = true;
        }
      });
    }

    child.once("error", () => {
      reject(new Error(`nao foi possivel iniciar ${command}`));
    });
    child.once("close", (code, signal) => {
      if (outputExceeded) {
        reject(new Error(`${command} excedeu o limite de saida capturada`));
        return;
      }

      const result = {
        code: code ?? 1,
        signal,
        stdout: Buffer.concat(stdout).toString("utf8"),
        stderr: Buffer.concat(stderr).toString("utf8"),
      };
      if (result.code !== 0 && !options.allowFailure) {
        reject(new Error(`${command} terminou com codigo ${result.code}`));
        return;
      }
      resolve(result);
    });
  });
}

export async function runNodeScript(rootDirectory, scriptName) {
  await runCommand(process.execPath, [`scripts/${scriptName}`], {
    cwd: rootDirectory,
    env: process.env,
  });
}
