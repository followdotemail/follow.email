import { NextResponse } from 'next/server';
import { SSHClient } from '@/lib/ssh';
import { DEMO_ENV_VARS } from '@/lib/demo';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
    const encoder = new TextEncoder();

    // Create a streaming response
    const customReadable = new ReadableStream({
        async start(controller) {
            const sendLog = (msg: string) => {
                controller.enqueue(encoder.encode(msg + '\n'));
                // console.log(msg); // Optional: keep server logs
            };

            const printHeader = (msg: string) => {
                sendLog('\n============================================================');
                sendLog(msg);
                sendLog('============================================================\n');
            };

            try {
                const body = await request.json();
                const { ip_address, private_key, repo_url, env_vars } = body;

                if (!ip_address || !private_key) {
                    sendLog("ERROR: Missing required fields");
                    controller.close();
                    return;
                }

                let cleanKey = private_key.trim();
                if (cleanKey.includes('\\n')) {
                    cleanKey = cleanKey.replace(/\\n/g, '\n');
                }
                cleanKey = cleanKey + '\n';

                const ssh = new SSHClient({
                    host: ip_address,
                    username: 'ubuntu',
                    privateKey: cleanKey
                });

                const repo = repo_url || process.env.GITHUB_REPO || 'https://github.com/followdotemail/follow.email.git';

                // 1. Deploy Application
                printHeader('Deploying Application');

                sendLog('Creating app directory...');
                await ssh.execute('sudo mkdir -p /opt/follow.email && sudo chown ubuntu:ubuntu /opt/follow.email');

                sendLog('Cloning repository...');
                const gitCheck = await ssh.execute('[ -d /opt/follow.email/.git ] && echo "EXISTS"');
                if (gitCheck.stdout.trim().includes('EXISTS')) {
                    sendLog('Repo exists, pulling latest...');
                    await ssh.execute('cd /opt/follow.email && git fetch --all && git reset --hard origin/$(git rev-parse --abbrev-ref HEAD)');
                } else {
                    sendLog(`Cloning ${repo}...`);
                    await ssh.execute(`git clone ${repo} /opt/follow.email`);
                }
                sendLog('[OK] Application deployment base setup complete!');

                // 2. Setup Env Vars
                printHeader('Setting Up Environment Variables');

                const localEnv: Record<string, string> = {};
                const safeKeyRegex = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

                // Load from process.env (filtered)
                for (const key in process.env) {
                    if (
                        !key.startsWith('npm_') &&
                        !key.startsWith('__') &&
                        !key.startsWith('NODE_') &&
                        !key.startsWith('NEXT_') &&
                        safeKeyRegex.test(key) &&
                        process.env[key]
                    ) {
                        localEnv[key] = process.env[key] as string;
                    }
                }

                // Merge: Local < Demo < Request
                // Force PORT=8080 to match docker-compose mapping (Host 8080 -> Container 8080)
                // The app defaults to 3000 if not set, causing 502 Bad Gateway.
                const finalEnv = { ...DEMO_ENV_VARS, ...localEnv, ...env_vars, PORT: '8080' };

                sendLog('Configuring environment variables...');
                let envContent = '';
                for (const [key, value] of Object.entries(finalEnv)) {
                    if (value) {
                        envContent += `${key}=${value}\n`;
                    }
                }

                const b64Env = Buffer.from(envContent).toString('base64');
                await ssh.execute(`echo "${b64Env}" | base64 -d > /opt/follow.email/.env`);
                await ssh.execute(`cp /opt/follow.email/.env /opt/follow.email/infra/.env`);

                sendLog('[OK] Environment variables configured!');

                // 3. Start Application
                printHeader('Starting Application');

                sendLog('Building Docker image (this may take a while)...');
                const buildCmd = 'cd /opt/follow.email/infra && sudo docker compose build backend';
                sendLog(`> ${buildCmd}`);

                // For long running commands, we might want to capture output progressively if SSHClient supported it.
                // Current SSHClient awaits full result. For "Step 7" waiting, at least getting "Building..." is better than silence.
                const buildRes = await ssh.execute(buildCmd);
                if (buildRes.stdout) sendLog(buildRes.stdout);
                if (buildRes.stderr) sendLog(buildRes.stderr);

                sendLog('Starting application...');
                const upCmd = 'cd /opt/follow.email/infra && sudo docker compose up -d backend';
                sendLog(`> ${upCmd}`);
                await ssh.execute(upCmd);

                sendLog('Waiting for app to be healthy (sleep 10s)...');
                await ssh.execute('sleep 10');

                sendLog('Checking container status...');
                const psRes = await ssh.execute('cd /opt/follow.email/infra && sudo docker compose ps backend');
                sendLog(psRes.stdout);

                sendLog('[OK] Application started successfully!');

            } catch (error: any) {
                console.error("Deploy Error", error);
                sendLog(`ERROR: ${error.message}`);
                if (error.logs) sendLog(error.logs);
            } finally {
                controller.close();
            }
        }
    });

    return new Response(customReadable, {
        headers: {
            'Content-Type': 'text/plain; charset=utf-8',
            'Transfer-Encoding': 'chunked',
        },
    });
}
