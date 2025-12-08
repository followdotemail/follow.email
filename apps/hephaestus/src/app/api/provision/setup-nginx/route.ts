import { NextResponse } from 'next/server';
import { SSHClient } from '@/lib/ssh';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
    try {
        const body = await request.json();
        const { ip_address, private_key, record_name } = body;

        if (!ip_address || !private_key || !record_name) {
            return NextResponse.json(
                { error: 'Missing required fields: ip_address, private_key, record_name' },
                { status: 400 }
            );
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

        let logs = '';
        const addLog = (msg: string) => { logs += msg + '\n'; };
        const printHeader = (msg: string) => {
            addLog('\n============================================================');
            addLog(msg);
            addLog('============================================================\n');
        };

        printHeader('Configuring Nginx');

        const domain = `${record_name}.follow.email`;

        // Exact template from provision-dev-instance.py
        const nginxConfig = `server {
    listen 80;
    server_name ${domain};

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Proxy settings
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Health check endpoint
    location /api/v1/health {
        proxy_pass http://localhost:8080;
        access_log off;
    }
}`;

        // 1. Write config to remote temp file
        addLog('Writing Nginx config to server...');
        const b64Config = Buffer.from(nginxConfig).toString('base64');
        await ssh.execute(`echo "${b64Config}" | base64 -d > /tmp/nginx-config.tmp`);

        // 2. Move to sites-available
        addLog('Moving config to sites-available...');
        await ssh.execute(`sudo mv -f /tmp/nginx-config.tmp /etc/nginx/sites-available/${record_name}.follow.email`);
        await ssh.execute(`sudo chmod 644 /etc/nginx/sites-available/${record_name}.follow.email`);

        // 3. Symlink to sites-enabled
        addLog('Enabling site...');
        await ssh.execute(`sudo ln -sf /etc/nginx/sites-available/${record_name}.follow.email /etc/nginx/sites-enabled/`);

        // 4. Remove default site (optional but good practice)
        await ssh.execute(`[ -f /etc/nginx/sites-enabled/default ] && sudo rm -f /etc/nginx/sites-enabled/default || true`);

        // 5. Test Config
        addLog('Testing Nginx configuration...');
        const testRes = await ssh.execute('sudo nginx -t');
        if (testRes.code !== 0) {
            logs += `ERR: ${testRes.stderr}\n`;
            throw new Error('Nginx configuration test failed');
        }

        // 6. Reload Nginx
        addLog('Reloading Nginx...');
        await ssh.execute('sudo systemctl reload nginx');

        // 7. Verify Port 80
        addLog('Verifying Nginx status...');
        const statusCheck = await ssh.execute("sudo systemctl is-active nginx && echo 'Nginx is active'");
        if (statusCheck.stdout.includes('Nginx is active')) {
            addLog('[OK] Nginx configured successfully!');
        } else {
            addLog('[WARN] Nginx might not be running correctly.');
        }

        return NextResponse.json({
            success: true,
            logs
        });

    } catch (error: any) {
        console.error('Nginx Setup Error:', error);
        return NextResponse.json(
            { error: error.message || 'Failed to setup Nginx', logs: error.logs },
            { status: 500 }
        );
    }
}
