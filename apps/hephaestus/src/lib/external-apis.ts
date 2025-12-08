
import axios from 'axios';
import https from 'https';

const agent = new https.Agent({ family: 4 });

// Excloud Types
interface CreateInstanceConfig {
    name: string;
    ssh_pubkey: string;
    zone_id?: number;
    subnet_id?: number;
    project_id?: number;
    security_group_ids?: number[];
    image_id?: number;
}

// Environment variables should be loaded securely
const EXCLOUD_API_KEY = process.env.EXCLOUD_API_KEY;
const EXCLOUD_BASE_URL = process.env.EXCLOUD_API_URL || 'https://compute.excloud.in';

const SPACESHIP_API_KEY = process.env.SPACESHIP_API_KEY;
const SPACESHIP_API_SECRET = process.env.SPACESHIP_API_SECRET;
const SPACESHIP_BASE_URL = 'https://spaceship.dev/api/v1';

export const ExcloudAPI = {
    async createInstance(config: CreateInstanceConfig) {
        if (!EXCLOUD_API_KEY) throw new Error('EXCLOUD_API_KEY is not set');

        // Config matched from provision-dev-instance.py
        const payload = {
            allocate_public_ipv4: true,
            image_id: config.image_id || 10, // Ubuntu 22.04 LTS as per script
            instance_type: "t1.medium",
            name: config.name,
            project_id: config.project_id || 1,
            ssh_pubkey: config.ssh_pubkey,
            subnet_id: config.subnet_id || 1273,
            zone_id: config.zone_id || 1,
            security_group_ids: config.security_group_ids || [1624],
            root_volume: {
                size_gib: 24,
                zone_id: config.zone_id || 1,
                baseline_iops: 3000,
                baseline_throughput_mbps: 250
            }
        };

        const response = await axios.post(`${EXCLOUD_BASE_URL}/compute/create`, payload, {
            httpsAgent: agent,
            headers: {
                'Authorization': `Bearer ${EXCLOUD_API_KEY}`,
                'Content-Type': 'application/json'
            }
        });

        return response.data;
    },

    async getInstanceStatus(vmId: number, zoneId: number = 1) {
        if (!EXCLOUD_API_KEY) throw new Error('EXCLOUD_API_KEY is not set');

        const response = await axios.get(`${EXCLOUD_BASE_URL}/compute/list?zone_id=${zoneId}`, {
            httpsAgent: agent,
            headers: {
                'Authorization': `Bearer ${EXCLOUD_API_KEY}`,
                'Content-Type': 'application/json'
            }
        });

        const vms = response.data;
        return vms.find((vm: any) => vm.vm_id === vmId);
    },

    async listInstances(zoneId: number = 1) {
        if (!EXCLOUD_API_KEY) throw new Error('EXCLOUD_API_KEY is not set');

        try {
            const response = await axios.get(`${EXCLOUD_BASE_URL}/compute/list?zone_id=${zoneId}`, {
                httpsAgent: agent,
                headers: {
                    'Authorization': `Bearer ${EXCLOUD_API_KEY}`,
                    'Content-Type': 'application/json'
                }
            });
            return response.data;
        } catch (error) {
            console.error("Failed to list instances:", error);
            return [];
        }
    },

    async terminateInstance(vmId: number) {
        if (!EXCLOUD_API_KEY) throw new Error('EXCLOUD_API_KEY is not set');

        const response = await axios.post(`${EXCLOUD_BASE_URL}/compute/terminate`, { vm_id: vmId }, {
            httpsAgent: agent,
            headers: {
                'Authorization': `Bearer ${EXCLOUD_API_KEY}`,
                'Content-Type': 'application/json'
            }
        });
        return response.data;
    }
};

export const SpaceshipAPI = {
    async updateARecord(domain: string, recordName: string, ipAddress: string) {
        if (!SPACESHIP_API_KEY || !SPACESHIP_API_SECRET) throw new Error('Spaceship credentials not set');

        const payload = {
            force: false,
            items: [
                {
                    address: ipAddress,
                    name: recordName,
                    type: "A",
                    ttl: 1800
                }
            ]
        };

        const response = await axios.put(`${SPACESHIP_BASE_URL}/dns/records/${domain}`, payload, {
            headers: {
                'X-API-Key': SPACESHIP_API_KEY,
                'X-API-Secret': SPACESHIP_API_SECRET,
                'Content-Type': 'application/json'
            }
        });

        return response.data;
    },

    async deleteARecord(domain: string, recordName: string, ipAddress: string) {
        if (!SPACESHIP_API_KEY || !SPACESHIP_API_SECRET) throw new Error('Spaceship credentials not set');

        // Logic from python script: specific payload structure for deletion
        const payload = [
            {
                type: "A",
                address: ipAddress,
                name: recordName
            }
        ];

        try {
            await axios.delete(`${SPACESHIP_BASE_URL}/dns/records/${domain}`, {
                headers: {
                    'X-API-Key': SPACESHIP_API_KEY,
                    'X-API-Secret': SPACESHIP_API_SECRET,
                    'Content-Type': 'application/json'
                },
                data: payload
            });
            return true;
        } catch (e: any) {
            // Enhance error message for debugging
            const msg = e.response?.data?.error || e.message || 'Unknown Spaceship API error';
            throw new Error(`Delete failed: ${msg}`);
        }
    },

    async getDnsRecords(domain: string) {
        if (!SPACESHIP_API_KEY || !SPACESHIP_API_SECRET) throw new Error('Spaceship credentials not set');

        // Match Python script query params to avoid 422 error
        const response = await axios.get(`${SPACESHIP_BASE_URL}/dns/records/${domain}?take=100&skip=0&orderBy=type`, {
            headers: {
                'X-API-Key': SPACESHIP_API_KEY,
                'X-API-Secret': SPACESHIP_API_SECRET,
                'Content-Type': 'application/json'
            }
        });

        return response.data.items || [];
    }
}
