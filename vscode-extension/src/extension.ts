import * as vscode from 'vscode';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import * as cp from 'child_process';

interface SessionEvent {
    type: 'prompt' | 'response';
    sessionId: string;
    timestamp: number;
}

const PROCESSED_SESSIONS = new Set<string>();
let outputChannel: vscode.OutputChannel;

export function activate(context: vscode.ExtensionContext) {
    outputChannel = vscode.window.createOutputChannel('TT Copilot Bridge');
    outputChannel.appendLine('TT Copilot Bridge activated');

    const watcher = createWorkspaceWatcher();
    if (watcher) {
        context.subscriptions.push(watcher);
    }

    // Initial scan of existing sessions
    scanExistingSessions();
}

export function deactivate() {
    outputChannel?.dispose();
}

function createWorkspaceWatcher(): vscode.FileSystemWatcher | null {
    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (!workspaceFolders) {
        return null;
    }

    // Watch for new transcript files across all workspaceStorage directories
    const watcher = vscode.workspace.createFileSystemWatcher('**/GitHub.copilot-chat/transcripts/*.jsonl');

    watcher.onDidCreate((uri) => {
        outputChannel.appendLine(`New transcript file: ${uri.fsPath}`);
        handleNewTranscript(uri.fsPath);
    });

    watcher.onDidChange((uri) => {
        outputChannel.appendLine(`Transcript changed: ${uri.fsPath}`);
        handleTranscriptChange(uri.fsPath);
    });

    return watcher;
}

function scanExistingSessions() {
    const storagePaths = getWorkspaceStoragePaths();
    for (const storagePath of storagePaths) {
        const chatDir = path.join(storagePath, 'GitHub.copilot-chat');
        if (!fs.existsSync(chatDir)) continue;

        const transcriptsDir = path.join(chatDir, 'transcripts');
        if (!fs.existsSync(transcriptsDir)) continue;

        const files = fs.readdirSync(transcriptsDir).filter(f => f.endsWith('.jsonl'));
        for (const file of files) {
            const filePath = path.join(transcriptsDir, file);
            const sessionId = path.basename(file, '.jsonl');
            if (!PROCESSED_SESSIONS.has(sessionId)) {
                handleNewTranscript(filePath);
            }
        }
    }
}

function handleNewTranscript(filePath: string) {
    const sessionId = path.basename(filePath, '.jsonl');
    if (PROCESSED_SESSIONS.has(sessionId)) return;

    PROCESSED_SESSIONS.add(sessionId);

    // Read the transcript to extract timestamps
    try {
        const content = fs.readFileSync(filePath, 'utf-8');
        const lines = content.split('\n').filter(l => l.trim());

        let promptTimestamp = 0;
        let responseTimestamp = 0;

        for (const line of lines) {
            try {
                const event = JSON.parse(line);
                if (event.type === 'user.message' && event.timestamp) {
                    promptTimestamp = new Date(event.timestamp).getTime();
                    callTtRecord('prompt', sessionId, promptTimestamp);
                } else if (event.type === 'assistant.message' && event.timestamp) {
                    responseTimestamp = new Date(event.timestamp).getTime();
                    callTtRecord('response', sessionId, responseTimestamp);
                }
            } catch {
                // Skip malformed lines
            }
        }
    } catch (err) {
        outputChannel.appendLine(`Error reading transcript: ${err}`);
    }
}

function handleTranscriptChange(filePath: string) {
    // For now, treat changes similar to new files
    handleNewTranscript(filePath);
}

function findTtBin(): string {
    // 1. Check TT_BIN env var
    const envBin = process.env.TT_BIN;
    if (envBin) {
        if (fs.existsSync(envBin)) return envBin;
        outputChannel.appendLine(`TT_BIN set to "${envBin}" but not found, falling back`);
    }

    // 2. Check common install locations
    const home = os.homedir();
    const candidates = [
        path.join(home, '.local', 'bin', 'tt'),
        path.join(home, 'bin', 'tt'),
        path.join(home, 'go', 'bin', 'tt'),
        '/usr/local/bin/tt',
        '/opt/homebrew/bin/tt',
    ];

    for (const candidate of candidates) {
        if (fs.existsSync(candidate)) return candidate;
    }

    // 3. Fallback — rely on PATH
    return 'tt';
}

function callTtRecord(type: 'prompt' | 'response', sessionId: string, timestamp: number) {
    const workspaceFolder = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath || process.cwd();
    const ttBin = findTtBin();

    const args = [
        'record', type,
        '--tool', 'vscode-copilot',
        '--session', sessionId,
        '--project', workspaceFolder,
    ];

    outputChannel.appendLine(`Calling: ${ttBin} ${args.join(' ')}`);

    try {
        const result = cp.spawnSync(ttBin, args, {
            encoding: 'utf-8',
            timeout: 5000,
        });

        if (result.error) {
            outputChannel.appendLine(`tt not found or error: ${result.error.message}`);
            outputChannel.appendLine('Set TT_BIN env var to the full path of tt, or install tt in ~/.local/bin/');
        } else if (result.status !== 0) {
            outputChannel.appendLine(`tt exited with status ${result.status}: ${result.stderr}`);
        }
    } catch (err) {
        outputChannel.appendLine(`Error calling tt: ${err}`);
    }
}

function getWorkspaceStoragePaths(): string[] {
    const paths: string[] = [];
    const home = os.homedir();
    const platform = process.platform;

    if (platform === 'darwin') {
        const base = path.join(home, 'Library', 'Application Support');
        paths.push(
            path.join(base, 'Code', 'User', 'workspaceStorage'),
            path.join(base, 'Code - Insiders', 'User', 'workspaceStorage'),
            path.join(base, 'VSCodium', 'User', 'workspaceStorage'),
            path.join(base, 'Cursor', 'User', 'workspaceStorage'),
        );
    } else if (platform === 'linux') {
        const base = path.join(home, '.config');
        paths.push(
            path.join(base, 'Code', 'User', 'workspaceStorage'),
            path.join(base, 'Code - Insiders', 'User', 'workspaceStorage'),
            path.join(base, 'VSCodium', 'User', 'workspaceStorage'),
            path.join(base, 'Cursor', 'User', 'workspaceStorage'),
        );
    } else if (platform === 'win32') {
        const base = process.env.APPDATA || path.join(home, 'AppData', 'Roaming');
        paths.push(
            path.join(base, 'Code', 'User', 'workspaceStorage'),
            path.join(base, 'Code - Insiders', 'User', 'workspaceStorage'),
            path.join(base, 'VSCodium', 'User', 'workspaceStorage'),
            path.join(base, 'Cursor', 'User', 'workspaceStorage'),
        );
    }

    return paths.filter(p => fs.existsSync(p));
}
