
'use client';

import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Terminal } from './terminal';
import { Loader2, CheckCircle2, XCircle, Plus, Key, RefreshCw, Trash2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog"

interface SSHKey {
    id: number;
    name: string;
    public_key: string;
    private_key: string;
}

export function ProvisioningOrchestrator() {
    // Configuration State
    const [config, setConfig] = useState({
        name: 'follow-email-backend',
        recordName: 'api',
        sshKey: '',
        sshPub: '',
    });

    // Instance List State
    const [instances, setInstances] = useState<any[]>([]);
    const [isLoadingInstances, setIsLoadingInstances] = useState(false);

    // SSH Key Management State
    const [sshKeys, setSshKeys] = useState<any[]>([]);
    const [selectedKeyId, setSelectedKeyId] = useState<string>('');
    const [isAddKeyOpen, setIsAddKeyOpen] = useState(false);
    const [newKey, setNewKey] = useState({ name: '', private: '', public: '' });
    const [isSavingKey, setIsSavingKey] = useState(false);

    // Execution State
    const [isProvisioning, setIsProvisioning] = useState(false);
    const [currentStep, setCurrentStep] = useState(0);
    const [logs, setLogs] = useState<string[]>([]);
    const [error, setError] = useState<string | null>(null);
    const [instanceInfo, setInstanceInfo] = useState<any>(null);

    const addLog = (msg: string) => setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] ${msg}`]);

    // Fetch Data on Load
    useEffect(() => {
        fetchKeys();
        fetchInstances();
    }, []);

    const fetchInstances = async () => {
        setIsLoadingInstances(true);
        try {
            const res = await axios.get('/api/instances');
            if (res.data.success) {
                setInstances(res.data.instances || []);
            }
        } catch (e) {
            console.error("Failed to fetch instances", e);
        } finally {
            setIsLoadingInstances(false);
        }
    };

    const fetchKeys = async () => {
        try {
            const res = await axios.get('/api/keys');
            setSshKeys(res.data);
        } catch (e) {
            console.error("Failed to fetch keys", e);
        }
    };

    const handleKeySelect = (keyId: string) => {
        setSelectedKeyId(keyId);
        const key = sshKeys.find(k => k.id.toString() === keyId);
        if (key) {
            setConfig(prev => ({
                ...prev,
                sshPub: key.public_key,
                sshKey: key.private_key || ''
            }));
        }
    };

    const handleSaveKey = async () => {
        setIsSavingKey(true);
        try {
            const res = await axios.post('/api/keys', {
                name: newKey.name,
                private_key: newKey.private,
                public_key: newKey.public
            });
            setSshKeys(prev => [res.data, ...prev]);
            setSelectedKeyId(res.data.id.toString());
            setConfig(prev => ({
                ...prev,
                sshPub: res.data.public_key,
                sshKey: res.data.private_key
            }));
            setIsAddKeyOpen(false);
            setNewKey({ name: '', private: '', public: '' });
        } catch (e) {
            console.error(e);
            alert("Failed to save key");
        } finally {
            setIsSavingKey(false);
        }
    };

    const handleTerminate = async (vmId: number) => {
        if (!confirm('Are you sure you want to terminate this instance? This action is irreversible.')) return;

        try {
            const res = await axios.delete('/api/instances', { data: { vm_id: vmId } });
            if (res.data.success) {
                await fetchInstances();
            }
        } catch (e) {
            console.error("Failed to terminate instance", e);
            alert('Failed to terminate instance');
        }
    };

    const handleProvision = async () => {
        if (!config.sshKey || !config.sshPub) {
            setError("SSH Keys are required!");
            return;
        }

        setIsProvisioning(true);
        setError(null);
        setLogs([]);
        addLog("Starting provisioning process...");
        setCurrentStep(1);

        try {
            // STEP 1: Create Instance
            addLog("Step 1: Creating Excloud Instance...");
            const createRes = await axios.post('/api/provision/create-instance', {
                name: config.name,
                ssh_pubkey: config.sshPub,
            });

            if (createRes.data.logs) {
                createRes.data.logs.split('\n').filter((l: string) => l.trim()).forEach((line: string) => addLog(line));
            }

            const { instance_id, instance_ip } = createRes.data;
            addLog(`Instance created! ID: ${instance_id}, IP: ${instance_ip}`);
            setInstanceInfo({ id: instance_id, ip: instance_ip });
            setCurrentStep(2);

            // STEP 2: Wait for Ready
            addLog("Step 2: Waiting for instance to be ready...");
            let attempts = 0;
            const maxAttempts = 30; // 5 minutes approx
            while (attempts < maxAttempts) {
                await new Promise(r => setTimeout(r, 10000)); // Wait 10s
                attempts++;

                try {
                    const statusRes = await axios.post('/api/provision/status', { vm_id: instance_id });
                    if (statusRes.data.is_ready) {
                        addLog("Instance is RUNNING and ready!");
                        break;
                    } else {
                        addLog(`Waiting... State: ${statusRes.data.state}`);
                    }
                } catch (e) {
                    addLog("Retrying status check...");
                }
            }
            if (attempts >= maxAttempts) throw new Error("Timeout waiting for instance ready state");
            setCurrentStep(3);

            // Give SSH a moment to wake up fully
            addLog("Waiting 30s for SSH to initialize...");
            await new Promise(r => setTimeout(r, 30000));

            // STEP 3: Update DNS
            addLog(`Step 3: Updating DNS for ${config.recordName}.follow.email...`);
            const dnsRes = await axios.post('/api/provision/setup-dns', {
                domain: 'follow.email',
                record_name: config.recordName,
                ip_address: instance_ip
            });
            if (dnsRes.data.logs) {
                dnsRes.data.logs.split('\n').filter((l: string) => l.trim()).forEach((line: string) => addLog(line));
            }
            addLog("DNS update step completed.");
            setCurrentStep(4);

            // STEP 4: Install Dependencies
            addLog("Step 4: Installing System Dependencies (Docker, Nginx)...");
            addLog("This may take 2-3 minutes. Do not close this window.");
            const installRes = await axios.post('/api/provision/install-dependencies', {
                ip_address: instance_ip,
                private_key: config.sshKey
            });
            addLog("Dependencies installed.");
            if (installRes.data.logs) {
                installRes.data.logs.split('\n').filter((l: string) => l.trim()).forEach((line: string) => addLog(line));
            }
            setCurrentStep(5);

            // STEP 5: Setup Nginx (HTTP Proxy)
            addLog("Step 5: Configuring Nginx Reverse Proxy...");
            const nginxRes = await axios.post('/api/provision/setup-nginx', {
                ip_address: instance_ip,
                private_key: config.sshKey,
                record_name: config.recordName
            });
            if (nginxRes.data.logs) {
                nginxRes.data.logs.split('\n').filter((l: string) => l.trim()).forEach((line: string) => addLog(line));
            }
            addLog("Nginx configured successfully.");
            setCurrentStep(6);

            // STEP 6: SSL Setup
            addLog("Step 6: Configuring SSL Certificates...");
            let sslSuccess = false;
            try {
                const sslRes = await axios.post('/api/provision/setup-ssl', {
                    ip_address: instance_ip,
                    private_key: config.sshKey,
                    domain: `${config.recordName}.follow.email`
                });
                addLog(`SSL Configured! Source: ${sslRes.data.source}`);
                sslSuccess = true;
            } catch (sslErr: any) {
                console.warn("SSL Setup Failed:", sslErr);
                addLog(`[WARN] SSL Setup Failed: ${sslErr.response?.data?.error || sslErr.message}`);
                addLog("[WARN] Continuing with HTTP only (just like Python script behavior).");
            }
            setCurrentStep(7);

            // STEP 7: Deploy App
            const protocol = sslSuccess ? 'https' : 'http';
            addLog(`Step 7: Deploying Application (${protocol})...`);

            // Use fetch for streaming logs
            const deployRes = await fetch('/api/provision/deploy', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    ip_address: instance_ip,
                    private_key: config.sshKey,
                    env_vars: {
                        BASE_URL: `${protocol}://${config.recordName}.follow.email`,
                        NODE_ENV: 'production'
                    }
                })
            });

            if (!deployRes.ok) {
                const errJson = await deployRes.json();
                throw new Error(errJson.error || 'Deploy failed');
            }

            if (deployRes.body) {
                const reader = deployRes.body.getReader();
                const decoder = new TextDecoder();

                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;

                    const chunk = decoder.decode(value);
                    const lines = chunk.split('\n').filter(line => line.trim() !== '');
                    lines.forEach(line => addLog(line));
                }
            }

            addLog("Application Deployed!");
            addLog("PROVISIONING COMPLETED SUCCESSFULLY!");
            setCurrentStep(8); // Done

        } catch (err: any) {
            console.error(err);
            const msg = err.response?.data?.error || err.message || "Unknown error";
            setError(msg);
            addLog(`ERROR: ${msg}`);
            if (err.response?.data?.logs) {
                addLog('--- Server Logs ---');
                err.response.data.logs.split('\n').filter((l: string) => l.trim()).forEach((line: string) => addLog(line));
            }
        } finally {
            setIsProvisioning(false);
        }
    };

    const handleTroubleshoot = async () => {
        if (!instanceInfo?.ip || !config.sshKey) return;
        addLog("\n--- Starting Diagnostics ---");
        try {
            const res = await axios.post('/api/provision/troubleshoot', {
                ip_address: instanceInfo.ip,
                private_key: config.sshKey
            });
            if (res.data.logs) {
                res.data.logs.split('\n').filter((l: string) => l.trim()).forEach((line: string) => addLog(line));
            }
            addLog("--- End Diagnostics ---");
        } catch (e: any) {
            addLog(`Diagnostics Failed: ${e.message}`);
        }
    };

    return (
        <div className="min-h-screen bg-black text-zinc-100 flex flex-col font-sans selection:bg-zinc-800">
            <div className="w-full h-screen flex flex-col">
                <Tabs defaultValue="instances" className="h-full flex flex-col">
                    <div className="px-6 pt-6 border-b border-zinc-900 bg-zinc-950/50">
                        <div className="flex items-center justify-between mb-4">
                            <div className="flex items-center space-x-2">
                                <div className="h-6 w-6 bg-gradient-to-tr from-white to-zinc-500 rounded-lg"></div>
                                <h1 className="text-xl font-bold tracking-tight">Hephaestus</h1>
                            </div>
                            <Button
                                variant="ghost"
                                size="sm"
                                className="text-zinc-400 hover:text-white"
                                onClick={async () => {
                                    await fetch('/api/auth/logout', { method: 'POST' });
                                    window.location.reload();
                                }}
                            >
                                Logout
                            </Button>
                        </div>
                        <TabsList className="bg-transparent w-full justify-start rounded-none p-0 h-10 space-x-6">
                            <TabsTrigger
                                value="instances"
                                className="rounded-none border-b-2 border-transparent data-[state=active]:border-white data-[state=active]:bg-transparent px-1 py-2 text-zinc-400 data-[state=active]:text-white h-full transition-none data-[state=active]:shadow-none"
                            >
                                Instances
                            </TabsTrigger>
                            <TabsTrigger
                                value="provision"
                                className="rounded-none border-b-2 border-transparent data-[state=active]:border-white data-[state=active]:bg-transparent px-1 py-2 text-zinc-400 data-[state=active]:text-white h-full transition-none data-[state=active]:shadow-none"
                            >
                                Provision
                            </TabsTrigger>
                        </TabsList>
                    </div>

                    <TabsContent value="instances" className="flex-1 overflow-auto p-0 m-0 data-[state=inactive]:hidden outline-none bg-black">
                        <div className="p-6">
                            <div className="flex justify-end mb-4">
                                <Button variant="outline" size="sm" onClick={() => fetchInstances()} className="bg-zinc-900 border-zinc-800 hover:bg-zinc-800 text-zinc-300">
                                    <RefreshCw className={cn("w-3 h-3 mr-2", isLoadingInstances && "animate-spin")} />
                                    Refresh
                                </Button>
                            </div>
                            <div className="border border-zinc-800 rounded-md bg-zinc-950/30">
                                <Table>
                                    <TableHeader>
                                        <TableRow className="border-zinc-800 hover:bg-transparent">
                                            <TableHead className="text-zinc-500 w-[100px]">ID</TableHead>
                                            <TableHead className="text-zinc-500">Name</TableHead>
                                            <TableHead className="text-zinc-500">IP Address</TableHead>
                                            <TableHead className="text-zinc-500">State</TableHead>
                                            <TableHead className="text-zinc-500 text-right">Actions</TableHead>
                                        </TableRow>
                                    </TableHeader>
                                    <TableBody>
                                        {isLoadingInstances && instances.length === 0 ? (
                                            <TableRow className="border-zinc-800 hover:bg-transparent">
                                                <TableCell colSpan={5} className="text-center py-8 text-zinc-500">Loading instances...</TableCell>
                                            </TableRow>
                                        ) : instances.length === 0 ? (
                                            <TableRow className="border-zinc-800 hover:bg-transparent">
                                                <TableCell colSpan={5} className="text-center py-8 text-zinc-500">No instances found.</TableCell>
                                            </TableRow>
                                        ) : (
                                            instances.map((instance) => (
                                                <TableRow key={instance.vm_id} className="border-zinc-800 hover:bg-zinc-900/50">
                                                    <TableCell className="font-mono text-zinc-400">{instance.vm_id}</TableCell>
                                                    <TableCell className="font-medium text-zinc-200">{instance.name}</TableCell>
                                                    <TableCell className="font-mono text-zinc-400">{instance.public_ipv4}</TableCell>
                                                    <TableCell>
                                                        <div className={cn(
                                                            "inline-flex items-center px-2 py-1 rounded-full text-xs font-medium border uppercase",
                                                            instance.state?.toLowerCase() === 'terminated' && "bg-red-500/10 text-red-500 border-red-500/20",
                                                            instance.state?.toLowerCase() === 'terminating' && "bg-yellow-500/10 text-yellow-500 border-yellow-500/20",
                                                            instance.state?.toLowerCase() === 'running' && "bg-green-500/10 text-green-500 border-green-500/20",
                                                            ['creating', 'starting', 'provisioning'].includes(instance.state?.toLowerCase()) && "bg-blue-500/10 text-blue-500 border-blue-500/20",
                                                            !['terminated', 'terminating', 'running', 'creating', 'starting', 'provisioning'].includes(instance.state?.toLowerCase()) && "bg-zinc-500/10 text-zinc-500 border-zinc-500/20"
                                                        )}>
                                                            {instance.state}
                                                        </div>
                                                    </TableCell>
                                                    <TableCell className="text-right">
                                                        <Button
                                                            variant="ghost"
                                                            size="sm"
                                                            className="h-8 text-red-500 hover:text-red-400 hover:bg-red-500/10 disabled:opacity-50 disabled:pointer-events-none"
                                                            onClick={() => handleTerminate(instance.vm_id)}
                                                            disabled={['terminated', 'terminating'].includes(instance.state?.toLowerCase())}
                                                        >
                                                            <Trash2 className="w-3 h-3 mr-1" />
                                                            Terminate
                                                        </Button>
                                                    </TableCell>
                                                </TableRow>
                                            ))
                                        )}
                                    </TableBody>
                                </Table>
                            </div>
                        </div>
                    </TabsContent>

                    <TabsContent value="provision" className="flex-1 flex flex-col md:flex-row m-0 p-0 data-[state=inactive]:hidden h-full outline-none">
                        {/* Config Panel */}
                        <div className="w-full md:w-[400px] border-r border-zinc-900 bg-zinc-950/30 flex flex-col h-full overflow-y-auto">
                            <div className="p-6 space-y-8">
                                {/* Environment Config */}
                                <div className="space-y-4">
                                    <h2 className="text-xs uppercase tracking-widest text-zinc-500 font-semibold mb-4">Environment</h2>

                                    <div className="space-y-3">
                                        <div className="space-y-1.5">
                                            <Label className="text-xs text-zinc-400">Instance Name</Label>
                                            <Input
                                                value={config.name}
                                                onChange={e => setConfig({ ...config, name: e.target.value })}
                                                className="bg-black border-zinc-800 text-zinc-200 focus-visible:ring-zinc-700 h-9 text-sm rounded-md"
                                                placeholder="e.g. follow-email-dev"
                                            />
                                        </div>

                                        <div className="space-y-1.5">
                                            <Label className="text-xs text-zinc-400">Subdomain</Label>
                                            <div className="flex items-center">
                                                <Input
                                                    value={config.recordName}
                                                    onChange={e => setConfig({ ...config, recordName: e.target.value })}
                                                    className="bg-black border-zinc-800 text-zinc-200 focus-visible:ring-zinc-700 h-9 text-sm rounded-l-md rounded-r-none border-r-0 text-right pr-1"
                                                    placeholder="api"
                                                />
                                                <div className="h-9 px-3 flex items-center bg-zinc-900 border border-zinc-800 rounded-r-md text-zinc-500 text-sm border-l-0 font-medium">
                                                    .follow.email
                                                </div>
                                            </div>
                                        </div>
                                    </div>

                                    <div className="h-px bg-zinc-900 w-full" />

                                    {/* Security Config */}
                                    <div className="space-y-4">
                                        <div className="flex items-center justify-between mb-4">
                                            <h2 className="text-xs uppercase tracking-widest text-zinc-500 font-semibold">Security Creds</h2>
                                            <Dialog open={isAddKeyOpen} onOpenChange={setIsAddKeyOpen}>
                                                <DialogTrigger asChild>
                                                    <Button variant="ghost" size="sm" className="h-6 px-2 text-xs text-zinc-400 hover:text-white hover:bg-zinc-800">
                                                        <Plus className="w-3 h-3 mr-1" /> Add Key
                                                    </Button>
                                                </DialogTrigger>
                                                <DialogContent className="bg-zinc-950 border-zinc-800 text-zinc-100">
                                                    <DialogHeader>
                                                        <DialogTitle>Add New SSH Key</DialogTitle>
                                                        <DialogDescription className="text-zinc-400">
                                                            Enter the details for your new SSH key.
                                                        </DialogDescription>
                                                    </DialogHeader>
                                                    <div className="space-y-4 py-4">
                                                        <div className="space-y-2">
                                                            <Label>Key Name</Label>
                                                            <Input
                                                                value={newKey.name}
                                                                onChange={e => setNewKey({ ...newKey, name: e.target.value })}
                                                                className="bg-zinc-900 border-zinc-800"
                                                                placeholder="e.g. dev-laptop"
                                                            />
                                                        </div>
                                                        <div className="space-y-2">
                                                            <Label>Public Key</Label>
                                                            <Input
                                                                value={newKey.public}
                                                                onChange={e => setNewKey({ ...newKey, public: e.target.value })}
                                                                className="bg-zinc-900 border-zinc-800 font-mono text-xs"
                                                                placeholder="ssh-rsa AAAA..."
                                                            />
                                                        </div>
                                                        <div className="space-y-2">
                                                            <Label>Private Key</Label>
                                                            <textarea
                                                                value={newKey.private}
                                                                onChange={e => setNewKey({ ...newKey, private: e.target.value })}
                                                                className="w-full h-32 bg-zinc-900 border border-zinc-800 rounded-md p-2 font-mono text-xs focus:outline-none focus:ring-1 focus:ring-zinc-700"
                                                                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                                                            />
                                                        </div>
                                                    </div>
                                                    <DialogFooter>
                                                        <Button variant="outline" onClick={() => setIsAddKeyOpen(false)} className="border-zinc-800 hover:bg-zinc-900 text-zinc-400">Cancel</Button>
                                                        <Button onClick={handleSaveKey} disabled={isSavingKey} className="bg-white text-black hover:bg-zinc-200">
                                                            {isSavingKey && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                                                            Save Key
                                                        </Button>
                                                    </DialogFooter>
                                                </DialogContent>
                                            </Dialog>
                                        </div>

                                        <Select value={selectedKeyId} onValueChange={handleKeySelect}>
                                            <SelectTrigger className="w-full bg-zinc-900 border-zinc-800 text-zinc-200 h-11">
                                                <SelectValue placeholder="Select SSH Key" />
                                            </SelectTrigger>
                                            <SelectContent className="bg-zinc-950 border-zinc-800">
                                                {sshKeys.map((key) => (
                                                    <SelectItem key={key.id} value={key.id.toString()} className="text-zinc-200 focus:bg-zinc-900 cursor-pointer">
                                                        <span className="font-medium mr-2 text-white">{key.name}</span>
                                                        <span className="text-xs text-zinc-500 font-mono truncate max-w-[150px] inline-block align-bottom opacity-50">
                                                            {key.created_at?.split('T')[0]}
                                                        </span>
                                                    </SelectItem>
                                                ))}
                                                {sshKeys.length === 0 && (
                                                    <div className="p-2 text-xs text-zinc-500 text-center">No keys found</div>
                                                )}
                                            </SelectContent>
                                        </Select>

                                        {config.sshPub && (
                                            <div className="p-3 bg-blue-500/10 border border-blue-500/20 rounded-md">
                                                <div className="flex items-start space-x-2">
                                                    <Key className="w-4 h-4 text-blue-400 mt-0.5" />
                                                    <div className="flex-1 overflow-hidden">
                                                        <p className="text-xs text-blue-200 font-medium mb-0.5">Key Selected</p>
                                                        <p className="text-[10px] text-blue-300 font-mono truncate">{config.sshPub.substring(0, 30)}...</p>
                                                    </div>
                                                </div>
                                            </div>
                                        )}
                                    </div>

                                    <div className="pt-4">
                                        <Button
                                            className="w-full h-12 bg-white text-black hover:bg-zinc-200 font-semibold"
                                            onClick={handleProvision}
                                            disabled={isProvisioning || !config.sshKey}
                                        >
                                            {isProvisioning ? (
                                                <>
                                                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                                    Provisioning...
                                                </>
                                            ) : (
                                                'Create & Provision Instance'
                                            )}
                                        </Button>
                                        {!config.sshKey && (
                                            <p className="text-[10px] text-red-500 text-center mt-2">Please select an SSH key to proceed</p>
                                        )}
                                    </div>
                                </div>
                            </div>
                        </div>

                        {/* Execution View */}
                        <div className="flex-1 flex flex-col h-full overflow-hidden bg-black">
                            {/* Steps Indicator */}
                            <div className="border-b border-zinc-900 bg-zinc-950/30 p-4">
                                <div className="flex items-center space-x-2 overflow-x-auto no-scrollbar mask-linear-fade">
                                    {['Excloud VM', 'DNS Record', 'Dependencies', 'Nginx Config', 'SSL Cert', 'Deploy App'].map((step, idx) => {
                                        let state = 'pending';
                                        if (currentStep > idx) state = 'done';
                                        if (currentStep === idx && isProvisioning) state = 'active';
                                        if (error && currentStep === idx) state = 'error';

                                        return (
                                            <div key={step} className={cn(
                                                "flex items-center space-x-2 px-3 py-1.5 rounded-full text-xs font-medium whitespace-nowrap transition-colors",
                                                state === 'pending' && "text-zinc-600 bg-zinc-900/40",
                                                state === 'active' && "text-blue-400 bg-blue-500/10 border border-blue-500/20 animate-pulse",
                                                state === 'done' && "text-green-400 bg-green-500/10 border border-green-500/20",
                                                state === 'error' && "text-red-400 bg-red-500/10 border border-red-500/20",
                                            )}>
                                                <div className="flex items-center space-x-2">
                                                    {state === 'done' && <CheckCircle2 className="w-3 h-3" />}
                                                    {state === 'active' && <Loader2 className="w-3 h-3 animate-spin" />}
                                                    {state === 'error' && <XCircle className="w-3 h-3" />}
                                                    <span>{step}</span>
                                                </div>
                                            </div>
                                        )
                                    })}
                                </div>
                            </div>

                            {/* Terminal Window */}
                            <div className="flex-1 min-h-0 relative">
                                <Terminal logs={logs} className="h-full border-zinc-800/60 shadow-none bg-black absolute inset-0" />
                            </div>
                        </div>
                    </TabsContent>
                </Tabs>
            </div>
        </div>
    );
}
