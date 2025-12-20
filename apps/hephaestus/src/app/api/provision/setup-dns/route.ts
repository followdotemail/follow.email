
import { NextResponse } from 'next/server';
import { SpaceshipAPI } from '@/lib/external-apis';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
    let logs = '';
    const addLog = (msg: string) => {
        logs += msg + '\n';
        console.log(msg); // Keep server logs too
    };

    try {
        const body = await request.json();
        const { domain, record_name, ip_address } = body;

        if (!domain || !record_name || !ip_address) {
            return NextResponse.json(
                { error: 'Missing required fields: domain, record_name, ip_address' },
                { status: 400 }
            );
        }

        const rootDomain = domain || 'follow.email';
        addLog(`Configuring DNS for ${record_name}.${rootDomain}...`);

        // 1. Fetch existing records
        addLog(`Fetching existing records for ${rootDomain}...`);
        try {
            const records = await SpaceshipAPI.getDnsRecords(rootDomain);
            addLog(`Found ${records.length} total records.`);

            // Filter strictly for 'A' records matching the name
            const existingRecords = records.filter((r: any) => r.name === record_name && r.type === 'A');

            if (existingRecords.length > 0) {
                addLog(`Found ${existingRecords.length} existing A record(s) for ${record_name}. Deleting...`);

                for (const record of existingRecords) {
                    addLog(`Deleting record pointing to ${record.address}...`);
                    try {
                        await SpaceshipAPI.deleteARecord(rootDomain, record_name, record.address);
                        addLog(`[OK] Deleted A record (${record.address})`);
                    } catch (delErr: any) {
                        addLog(`[WARN] Failed to delete A record (${record.address}): ${delErr.message}`);
                    }
                }
            } else {
                addLog(`No existing A records found for ${record_name}.`);
            }
        } catch (e: any) {
            addLog(`[WARN] Failed to check/cleanup old DNS records: ${e.message}`);
            // Proceed anyway
        }

        // 2. Create/Update new record
        addLog(`Creating new A record: ${record_name} -> ${ip_address}`);
        try {
            await SpaceshipAPI.updateARecord(rootDomain, record_name, ip_address);
            addLog(`[OK] DNS updated successfully: ${record_name}.${rootDomain} -> ${ip_address}`);
        } catch (e: any) {
            addLog(`[ERR] Failed to update A record: ${e.message}`);
            throw e;
        }

        return NextResponse.json({
            success: true,
            message: `DNS updated: ${record_name}.${rootDomain} -> ${ip_address}`,
            logs
        });
    } catch (error: any) {
        console.error('DNS Update Error:', error);
        return NextResponse.json(
            { error: error.message || 'Failed to update DNS', logs },
            { status: 500 }
        );
    }
}
