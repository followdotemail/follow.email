import { NextResponse } from 'next/server';
import { SSHClient } from '@/lib/ssh';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
    try {
        const body = await request.json();
        const { ip_address, private_key } = body;

        if (!ip_address || !private_key) {
            return NextResponse.json({ error: 'Missing IP or Key' }, { status: 400 });
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

        let output = '';
        const addOut = (header: string, content: string) => {
            output += `\n=== ${header} ===\n${content}\n`;
        };

        // 1. Docker Status
        try {
            const ps = await ssh.execute('sudo docker ps -a');
            addOut('Docker Processes', ps.stdout);
        } catch (e: any) { addOut('Docker PS Failed', e.message); }

        // 2. Backend Logs
        try {
            const logs = await ssh.execute('cd /opt/follow.email/infra && sudo docker compose logs backend --tail=50');
            addOut('Backend Logs (Last 50)', logs.stdout + logs.stderr);
        } catch (e: any) { addOut('Backend Logs Failed', e.message); }

        // 3. Listening Ports
        try {
            // Install net-tools if missing? Or use ss
            const ports = await ssh.execute('sudo ss -tulpn');
            addOut('Listening Ports', ports.stdout);
        } catch (e: any) { addOut('Ports Check Failed', e.message); }

        // 4. Nginx Status
        try {
            const nginx = await ssh.execute('sudo systemctl status nginx --no-pager');
            addOut('Nginx Status', nginx.stdout);
        } catch (e: any) { addOut('Nginx Check Failed', e.message); }

        // 5. UFW Firewall Status
        try {
            // Check if UFW is active and what rules are present
            const ufw = await ssh.execute('sudo ufw status verbose');
            addOut('UFW Firewall Status', ufw.stdout);
        } catch (e: any) { addOut('UFW Check Failed', e.message); }

        return NextResponse.json({ success: true, logs: output });

    } catch (error: any) {
        return NextResponse.json({ error: error.message }, { status: 500 });
    }
}
