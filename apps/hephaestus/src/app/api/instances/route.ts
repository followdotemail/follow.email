
import { NextResponse } from 'next/server';
import { ExcloudAPI } from '@/lib/external-apis';

export const dynamic = 'force-dynamic';

export async function GET() {
    try {
        const instances = await ExcloudAPI.listInstances();
        return NextResponse.json({ success: true, instances });
    } catch (error: any) {
        console.error('List Instances Error:', error);
        return NextResponse.json(
            { error: error.message || 'Failed to list instances' },
            { status: 500 }
        );
    }
}

export async function DELETE(request: Request) {
    try {
        const body = await request.json();
        const { vm_id } = body;

        if (!vm_id) {
            return NextResponse.json({ error: 'Missing vm_id' }, { status: 400 });
        }

        const result = await ExcloudAPI.terminateInstance(vm_id);
        return NextResponse.json({ success: true, result });
    } catch (error: any) {
        console.error('Terminate Instance Error:', error);
        return NextResponse.json(
            { error: error.message || 'Failed to terminate instance' },
            { status: 500 }
        );
    }
}
