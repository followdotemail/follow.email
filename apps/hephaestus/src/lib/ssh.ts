import { Client, ClientChannel } from 'ssh2';

export interface SSHConfig {
    host: string;
    username: string;
    privateKey: string;
}

export interface CommandResult {
    stdout: string;
    stderr: string;
    code: number | null;
}

export class SSHClient {
    private config: SSHConfig;

    constructor(config: SSHConfig) {
        this.config = config;
    }

    execute(command: string): Promise<CommandResult> {
        return new Promise((resolve, reject) => {
            const conn = new Client();
            let stdout = '';
            let stderr = '';

            conn.on('ready', () => {
                conn.exec(command, (err: Error | undefined, stream: ClientChannel) => {
                    if (err) {
                        conn.end();
                        return reject(err);
                    }

                    stream.on('close', (code: number, signal: any) => {
                        conn.end();
                        resolve({ stdout, stderr, code });
                    }).on('data', (data: Buffer) => {
                        stdout += data.toString();
                    }).stderr.on('data', (data: Buffer) => {
                        stderr += data.toString();
                    });
                });
            }).on('error', (err: Error) => {
                reject(err);
            }).connect({
                host: this.config.host,
                port: 22,
                username: this.config.username,
                privateKey: this.config.privateKey,
                readyTimeout: 20000,
            });
        });
    }

    async testConnection(): Promise<boolean> {
        try {
            await this.execute('echo "SSH Ready"');
            return true;
        } catch (error) {
            console.error("SSH Connection failed:", error);
            return false;
        }
    }
}
