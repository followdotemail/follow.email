import { NextResponse } from 'next/server';
import { SSHClient } from '@/lib/ssh';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
    try {
        const body = await request.json();
        const { ip_address, private_key } = body;

        if (!ip_address || !private_key) {
            return NextResponse.json(
                { error: 'Missing required fields: ip_address, private_key' },
                { status: 400 }
            );
        }

        // Ensure private key has correct newlines
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

        if (!await ssh.testConnection()) {
            return NextResponse.json({ error: 'SSH connection failed' }, { status: 400 });
        }

        let logs = '';
        const addLog = (msg: string) => { logs += msg + '\n'; };

        // Formatting helpers to match Python Color/Header style (approximated in text)
        const printHeader = (msg: string) => {
            addLog('\n============================================================');
            addLog(msg);
            addLog('============================================================\n');
        };

        printHeader('Installing Dependencies');

        // Commands exactly from provision-dev-instance.py
        const steps = [
            { desc: "Updating system packages", cmd: "sudo apt-get update -y" },
            { desc: "Installing prerequisites", cmd: "sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common gnupg lsb-release" },
            { desc: "Adding Docker GPG key", cmd: "curl -fsSL --max-time 60 https://download.docker.com/linux/ubuntu/gpg | sudo gpg --batch --yes --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg" },
            { desc: "Adding Docker repository", cmd: "sudo bash -c 'echo \"deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable\" > /etc/apt/sources.list.d/docker.list'" },
            { desc: "Updating package index", cmd: "sudo apt-get update -y" },
            { desc: "Installing Docker", cmd: "sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin" },
            { desc: "Starting Docker service", cmd: "sudo systemctl start docker && sudo systemctl enable docker" },
            { desc: "Installing nginx", cmd: "sudo apt-get install -y nginx" },
            { desc: "Enabling nginx", cmd: "sudo systemctl enable nginx" },
            { desc: "Installing utilities", cmd: "sudo apt-get install -y git curl wget htop vim" }
        ];

        for (const step of steps) {
            addLog(`${step.desc}...`);
            // Only show output if error occurs or if explicitly needed (we default to hidden for clean logs)

            const res = await ssh.execute(step.cmd);

            if (res.code !== 0) {
                // Append output only on error
                if (res.stdout) logs += res.stdout + '\n';
                if (res.stderr) logs += `ERR: ${res.stderr}\n`;
                throw new Error(`Command failed: ${step.desc}\n${res.stderr}`);
            }

            addLog(`[OK] ${step.desc} completed`);
        }

        printHeader('Configuring Nginx');

        // Basic Nginx Verification
        addLog('Verifying nginx status...');
        const nginxCheck = await ssh.execute("sudo systemctl is-active nginx && echo 'Nginx is active'");
        if (nginxCheck.stdout.includes('Nginx is active')) {
            addLog('[OK] Nginx is running');
        } else {
            addLog('[WARN] Nginx status check failed');
        }

        return NextResponse.json({
            success: true,
            logs
        });

    } catch (error: any) {
        console.error('Install Deps Error:', error);
        return NextResponse.json(
            { error: error.message || 'Failed to install dependencies', logs: error.logs },
            { status: 500 }
        );
    }
}
