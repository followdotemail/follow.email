
import { NextResponse } from 'next/server';
import { ExcloudAPI } from '@/lib/external-apis';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
    try {
        const body = await request.json();
        const { name, zone_id, project_id, subnet_id, image_id } = body;

        // User request: Force usage of environment variable if available, ignoring body if so.
        // "the SSh pubkey should be taken from the environment variable"
        const final_ssh_pubkey = process.env.EXCLOUD_SSH_PUBKEY || body.ssh_pubkey;

        if (!name || !final_ssh_pubkey) {
            return NextResponse.json(
                { error: 'Missing required fields: name, ssh_pubkey (or EXCLOUD_SSH_PUBKEY env var)' },
                { status: 400 }
            );
        }

        let logs = '';
        const addLog = (msg: string) => { logs += msg + '\n'; };
        const printHeader = (msg: string) => {
            addLog('\n============================================================');
            addLog(msg);
            addLog('============================================================\n');
        };

        printHeader('Follow.Email Backend Instance Provisioning');

        // Log Environment Variables (Safe subset)
        addLog('\nEnvironment Variables Loaded:');
        addLog(`  EXCLOUD_IMAGE_ID: ${image_id || 10}`);
        addLog(`  EXCLOUD_ZONE_ID: ${zone_id || 1}`);
        addLog(`  EXCLOUD_SUBNET_ID: ${subnet_id || 1273}`);
        addLog(`  EXCLOUD_PROJECT_ID: ${project_id || 1}`);

        const envKey = process.env.EXCLOUD_SSH_PUBKEY;
        const maskedEnvKey = envKey ? `${envKey.substring(0, 10)}...` : 'Not Set';
        addLog(`  EXCLOUD_SSH_PUBKEY (ENV): ${maskedEnvKey}`);

        const usedKeyMasked = final_ssh_pubkey ? `${final_ssh_pubkey.substring(0, 10)}...` : 'None';
        addLog(`  Effective SSH Key: ${usedKeyMasked}`);

        addLog('\nCreating Excloud instance...');
        addLog('Sending request to: https://compute.excloud.in/compute/create');

        const payload = {
            name,
            ssh_pubkey: final_ssh_pubkey,
            zone_id: zone_id ? parseInt(zone_id) : 1,
            project_id: project_id ? parseInt(project_id) : 1,
            subnet_id: subnet_id ? parseInt(subnet_id) : 1273,
            image_id: image_id ? parseInt(image_id) : 10,
            allocate_public_ipv4: true,
            instance_type: "t1.medium",
            root_volume: {
                size_gib: 24,
                zone_id: zone_id ? parseInt(zone_id) : 1,
                baseline_iops: 3000,
                baseline_throughput_mbps: 250
            }
        };

        // Pretty print payload
        addLog(`Payload: ${JSON.stringify(payload, null, 2)}`);

        const instance = await ExcloudAPI.createInstance({
            name,
            ssh_pubkey: final_ssh_pubkey,
            zone_id: zone_id ? parseInt(zone_id) : undefined,
            project_id: project_id ? parseInt(project_id) : undefined,
            subnet_id: subnet_id ? parseInt(subnet_id) : undefined,
            image_id: image_id ? parseInt(image_id) : undefined,
        });

        addLog('[OK] Instance created successfully!');
        addLog(`  VM ID: ${instance.vm_id}`);
        addLog(`  Instance Type: ${instance.instance_type}`);
        addLog(`  IP Address: ${instance.public_ipv4}`);
        addLog(`  State: ${instance.state}`);

        return NextResponse.json({
            success: true,
            instance_id: instance.vm_id,
            instance_ip: instance.public_ipv4,
            data: instance,
            logs
        });
    } catch (error: any) {
        console.error('Create Instance Error:', error);
        return NextResponse.json(
            { error: error.message || 'Failed to create instance' },
            { status: 500 }
        );
    }
}
