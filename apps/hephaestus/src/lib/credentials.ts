
import { db } from './db';

export interface SSHKey {
    id: number;
    name: string;
    private_key: string;
    public_key: string;
    created_at: string;
}

export interface SSLCertificate {
    id: number;
    domain: string;
    fullchain: string;
    privkey: string;
    chain: string;
    cert: string;
    created_at: string;
    updated_at: string;
    expires_at: string;
}

export async function getSSHKeys(): Promise<SSHKey[]> {
    const res = await db.query('SELECT * FROM ssh_keys ORDER BY created_at DESC');
    return res.rows;
}

export async function getSSLCertificates(): Promise<SSLCertificate[]> {
    const res = await db.query('SELECT * FROM ssl_certificates ORDER BY created_at DESC');
    return res.rows;
}
