import { NextResponse } from 'next/server';
import { SSHClient } from '@/lib/ssh';
import { getSSLCertificates } from '@/lib/credentials';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
    try {
        const body = await request.json();
        const { ip_address, private_key, domain, email } = body;

        if (!ip_address || !private_key || !domain) {
            return NextResponse.json(
                { error: 'Missing required fields' },
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

        const rootDomain = domain; // e.g. api.follow.email
        const userEmail = email || 'admin@follow.email';
        const recordName = rootDomain.split('.')[0]; // 'api'

        let logs = '';
        const addLog = (msg: string) => { logs += msg + '\n'; };
        const printHeader = (msg: string) => {
            addLog('\n============================================================');
            addLog(msg);
            addLog('============================================================\n');
        };

        printHeader(`Setting up SSL Certificate from - Let's Encrypt Certbot for ${recordName}`);

        // 1. Check DB Variables
        const certs = await getSSLCertificates();
        const demoCert = certs.find(c => c.domain === rootDomain);

        let foundCert = null;
        if (demoCert && demoCert.fullchain && demoCert.privkey) {
            foundCert = demoCert;
        }

        if (foundCert) {
            printHeader(`Installing Existing SSL Certificate for ${rootDomain}`);
            addLog(`Found existing SSL certificates, uploading to server...`);

            // Create directories
            await ssh.execute(`sudo mkdir -p /etc/letsencrypt/live/${rootDomain} && sudo mkdir -p /etc/letsencrypt/archive/${rootDomain}`);

            const writeRemoteFile = async (content: string, path: string) => {
                const b64 = Buffer.from(content).toString('base64');
                await ssh.execute(`echo "${b64}" | base64 -d | sudo tee ${path} > /dev/null && sudo chmod 600 ${path}`);
            };

            await writeRemoteFile(foundCert.fullchain, `/etc/letsencrypt/live/${rootDomain}/fullchain.pem`);
            await writeRemoteFile(foundCert.privkey, `/etc/letsencrypt/live/${rootDomain}/privkey.pem`);

            // Update configuration to listen on SSL (Port 443) - Mirroring Python script
            // Added http2 to align exactly with Python script
            const nginxSslConfig = `server {
    listen 80;
    server_name ${rootDomain};
    
    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name ${rootDomain};

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/${rootDomain}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${rootDomain}/privkey.pem;
    
    # SSL Settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

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
            addLog(`Updating Nginx config to enable SSL...`);
            const b64Config = Buffer.from(nginxSslConfig).toString('base64');
            await ssh.execute(`echo "${b64Config}" | base64 -d > /tmp/nginx-ssl-config.tmp`);
            // Overwrite the existing HTTP-only config
            await ssh.execute(`sudo mv -f /tmp/nginx-ssl-config.tmp /etc/nginx/sites-available/${recordName}.follow.email`);
            await ssh.execute(`sudo chmod 644 /etc/nginx/sites-available/${recordName}.follow.email`);

            addLog(`[OK] Uploaded certificate files and updated Nginx config`);

            await ssh.execute(`sudo chmod 644 /etc/nginx/sites-available/${recordName}.follow.email`);

            // Ensure symlink exists (in case steps ran out of order)
            await ssh.execute(`sudo ln -sf /etc/nginx/sites-available/${recordName}.follow.email /etc/nginx/sites-enabled/`);

            // Validate Config
            addLog('Testing Nginx SSL configuration...');
            const testRes = await ssh.execute('sudo nginx -t');
            if (testRes.code !== 0) {
                logs += `ERR: Nginx Config Test Failed:\n${testRes.stderr}\n`;
                // Revert to non-ssl if possible? Or just throw.
                throw new Error('Invalid Nginx SSL Configuration generated');
            }
            addLog('[OK] Uploaded certificate files and updated Nginx config');

        } else {
            addLog(`Installing certbot...`);
            await ssh.execute('sudo apt-get install -y certbot python3-certbot-nginx');

            addLog(`Obtaining SSL certificate...`);
            const certbotCmd = `sudo certbot --nginx -d ${rootDomain} --non-interactive --agree-tos --email ${userEmail} --redirect`;
            addLog(`> ${certbotCmd}`);

            const certRes = await ssh.execute(certbotCmd);
            if (certRes.stdout) logs += certRes.stdout + '\n';
            if (certRes.stderr) logs += `ERR: ${certRes.stderr}\n`;

            if (certRes.code !== 0) {
                addLog(`[WARN] Warning: SSL certificate installation failed`);
                // Proceed anyway as per python script warning logic? Or fail?
                // Python script returns False but logic here might need to stay robust.
                // We'll throw to be safe for now, as web UI needs to know.
                throw new Error(`Certbot failed. Ensure DNS is propagated.\n${certRes.stderr}`);
            }
        }

        // Reload nginx
        addLog(`Reloading nginx to apply SSL configuration...`);
        const reloadRes = await ssh.execute('sudo systemctl reload nginx');
        if (reloadRes.code !== 0) {
            throw new Error(`Failed to reload Nginx: ${reloadRes.stderr}`);
        }
        addLog(`[OK] Nginx reloaded successfully`);

        addLog(`[OK] SSL certificate installed successfully!`);

        return NextResponse.json({
            success: true,
            logs,
            source: foundCert ? 'hardcoded' : 'certbot'
        });
    } catch (error: any) {
        console.error('SSL Setup Error:', error);
        return NextResponse.json(
            { error: error.message || 'Failed to setup SSL', logs: error.logs },
            { status: 500 }
        );
    }
}
