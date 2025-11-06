#!/usr/bin/env python3
"""
Follow.Email Development Instance Provisioning Script
======================================================
This script automates:
1. Creating a new Excloud instance with specified configuration
2. Installing Docker, nginx, and dependencies
3. Updating DNS records in Spaceship for api.follow.email
4. Setting up nginx reverse proxy
5. Deploying the Follow Email Backend application

Excloud API Documentation:
    https://excloud.in/docs/api/compute/

Configuration:
    All configuration should be in the root .env file of the repository.
    Required environment variables:
    - EXCLOUD_API_KEY
    - EXCLOUD_ZONE_ID
    - EXCLOUD_SUBNET_ID  
    - EXCLOUD_PROJECT_ID
    - EXCLOUD_SECURITY_GROUP_ID
    - EXCLOUD_SSH_PUBKEY
    - SPACESHIP_API_KEY
    - SPACESHIP_API_SECRET
    - DATABASE_URL (for application)
    - Other application environment variables
    
Usage:
    # From the deployments directory
    cd apps/backend/deployments
    python provision-dev-instance.py --action create --record-name api
    python provision-dev-instance.py --action destroy --vm-id 1234
    python provision-dev-instance.py --action setup-only
"""

import os
import sys
import time
import json
import argparse
import subprocess
import base64
from typing import Dict, Optional, List
import requests
from dotenv import load_dotenv

# Load environment variables from root .env file
env_path = os.path.join(os.path.dirname(__file__), '../../../.env')
load_dotenv(env_path)
print(f"Loading environment from: {os.path.abspath(env_path)}")


class Colors:
    """ANSI color codes for terminal output"""
    HEADER = '\033[95m'
    OKBLUE = '\033[94m'
    OKCYAN = '\033[96m'
    OKGREEN = '\033[92m'
    WARNING = '\033[93m'
    FAIL = '\033[91m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'


class ExcloudInstanceManager:
    """Manages Excloud instance provisioning"""
    
    def __init__(self):
        self.api_key = os.getenv('EXCLOUD_API_KEY')
        self.base_url = os.getenv('EXCLOUD_API_URL', 'https://compute.excloud.in')
        
        if not self.api_key:
            raise ValueError("EXCLOUD_API_KEY must be set in root .env file")
        
        self.headers = {
            'Authorization': f'Bearer {self.api_key}',
            'Content-Type': 'application/json'
        }
    
    def create_instance(self, config: Dict) -> Dict:
        """
        Create a new Excloud instance
        
        API: POST https://compute.excloud.in/compute/create
        Docs: https://excloud.in/docs/api/compute/#/%2Fcompute/compute-ComputeCreate
        
        Example Body:
        {
            "allocate_public_ipv4": true,
            "image_id": 10,
            "instance_type": "t1.medium",
            "name": "follow-email-backend",
            "project_id": 1,
            "ssh_pubkey": "",
            "subnet_id": 1273,
            "zone_id": 1,
            "security_group_ids": [1624],
            "root_volume": {
                "size_gib": 24,
                "zone_id": 1,
                "baseline_iops": 3000,
                "baseline_throughput_mbps": 250
            }
        }
        """
        print(f"{Colors.OKBLUE}Creating Excloud instance...{Colors.ENDC}")
        
        # Use the exact payload structure from the example
        payload = {
            "allocate_public_ipv4": config.get("allocate_public_ipv4", True),
            "image_id": config.get("image_id", 10),
            "instance_type": config.get("instance_type", "t1.medium"),
            "name": config.get("name", "follow-email-backend"),
            "project_id": config.get("project_id"),
            "ssh_pubkey": config.get("ssh_pubkey"),
            "subnet_id": config.get("subnet_id"),
            "zone_id": config.get("zone_id"),
            "security_group_ids": config.get("security_group_ids", []),
            "root_volume": config.get("root_volume", {
                "size_gib": 24,
                "zone_id": config.get("zone_id"),
                "baseline_iops": 3000,
                "baseline_throughput_mbps": 250
            })
        }
        
        try:
            print(f"{Colors.OKBLUE}Sending request to: {self.base_url}/compute/create{Colors.ENDC}")
            print(f"{Colors.OKBLUE}Payload: {json.dumps(payload, indent=2)}{Colors.ENDC}")
            
            response = requests.post(
                f"{self.base_url}/compute/create",
                headers=self.headers,
                json=payload,
                timeout=30
            )
            response.raise_for_status()
            instance_data = response.json()
            
            print(f"{Colors.OKGREEN}[OK] Instance created successfully!{Colors.ENDC}")
            print(f"  VM ID: {instance_data.get('vm_id')}")
            print(f"  Instance Type: {instance_data.get('instance_type')}")
            print(f"  IP Address: {instance_data.get('public_ipv4')}")
            print(f"  State: {instance_data.get('state')}")
            
            return instance_data
        
        except requests.exceptions.RequestException as e:
            print(f"{Colors.FAIL}[ERROR] Failed to create instance: {e}{Colors.ENDC}")
            if hasattr(e, 'response') and e.response is not None:
                print(f"  Response Status: {e.response.status_code}{Colors.ENDC}")
                print(f"  Response Body: {e.response.text}{Colors.ENDC}")
            raise
    
    def get_instance_status(self, vm_id: int, zone_id: int = None) -> Dict:
        """
        Get instance status
        
        API: GET https://compute.excloud.in/compute/list?zone_id=<zone_id>
        Docs: https://excloud.in/docs/api/compute/#/%2Fcompute/compute-ComputeList
        
        Note: zone_id is required for listing computes. Returns list of VMs, filter by vm_id.
        
        Example:
            GET /compute/list?zone_id=1
            Authorization: Bearer YOUR_API_KEY
        """
        try:
            # Add zone_id as query parameter if provided
            url = f"{self.base_url}/compute/list"
            if zone_id:
                url += f"?zone_id={zone_id}"
            
            response = requests.get(
                url,
                headers=self.headers,
                timeout=30
            )
            response.raise_for_status()
            vms = response.json()
            
            # Find the VM by ID
            for vm in vms:
                if vm.get('vm_id') == vm_id:
                    return vm
            
            raise ValueError(f"VM with ID {vm_id} not found in zone {zone_id if zone_id else 'any'}")
        except requests.exceptions.RequestException as e:
            print(f"{Colors.FAIL}[ERROR] Failed to get instance status: {e}{Colors.ENDC}")
            if hasattr(e, 'response') and e.response is not None:
                print(f"  Response Status: {e.response.status_code}{Colors.ENDC}")
                print(f"  Response Body: {e.response.text}{Colors.ENDC}")
            raise
    
    def wait_for_instance_ready(self, vm_id: int, zone_id: int = None, timeout: int = 300) -> bool:
        """
        Wait for instance to be in running state
        
        This function polls the Excloud API to check when the VM is ready to accept SSH connections.
        After creating an instance, it takes time for the OS to boot and SSH to start.
        """
        print(f"{Colors.OKBLUE}Waiting for instance to be ready...{Colors.ENDC}")
        
        start_time = time.time()
        while time.time() - start_time < timeout:
            try:
                status = self.get_instance_status(vm_id, zone_id)
                state = status.get('state', 'unknown')
                
                print(f"  Current state: {state}")
                
                if state.lower() in ['running', 'active']:
                    print(f"{Colors.OKGREEN}[OK] Instance is ready!{Colors.ENDC}")
                    return True
                elif state.lower() in ['failed', 'terminated', 'error']:
                    print(f"{Colors.FAIL}[ERROR] Instance failed to start{Colors.ENDC}")
                    return False
                
                time.sleep(10)
            except Exception as e:
                print(f"{Colors.WARNING}Warning: {e}{Colors.ENDC}")
                time.sleep(10)
        
        print(f"{Colors.FAIL}[ERROR] Timeout waiting for instance{Colors.ENDC}")
        return False
    
    def delete_instance(self, vm_id: int) -> bool:
        """
        Delete/terminate an instance
        
        API: POST https://compute.excloud.in/compute/terminate
        Docs: https://excloud.in/docs/api/compute/#/%2Fcompute/compute-ComputeTerminate
        
        Example Body:
        {
            "vm_id": 5856
        }
        
        Authorization: Bearer YOUR_API_KEY
        """
        print(f"{Colors.OKBLUE}Terminating instance {vm_id}...{Colors.ENDC}")
        
        try:
            payload = {"vm_id": vm_id}
            
            print(f"{Colors.OKBLUE}Sending request to: {self.base_url}/compute/terminate{Colors.ENDC}")
            print(f"{Colors.OKBLUE}Payload: {json.dumps(payload)}{Colors.ENDC}")
            
            response = requests.post(
                f"{self.base_url}/compute/terminate",
                headers=self.headers,
                json=payload,
                timeout=30
            )
            response.raise_for_status()
            print(f"{Colors.OKGREEN}[OK] Instance terminated successfully{Colors.ENDC}")
            return True
        except requests.exceptions.RequestException as e:
            print(f"{Colors.FAIL}[ERROR] Failed to terminate instance: {e}{Colors.ENDC}")
            if hasattr(e, 'response') and e.response is not None:
                print(f"  Response Status: {e.response.status_code}{Colors.ENDC}")
                print(f"  Response Body: {e.response.text}{Colors.ENDC}")
            return False


class SpaceshipDNSManager:
    """Manages DNS records via Spaceship API"""
    
    def __init__(self):
        self.api_key = os.getenv('SPACESHIP_API_KEY')
        self.api_secret = os.getenv('SPACESHIP_API_SECRET')
        self.domain = 'follow.email'
        self.base_url = 'https://spaceship.dev/api/v1'
        
        if not self.api_key or not self.api_secret:
            raise ValueError("SPACESHIP_API_KEY and SPACESHIP_API_SECRET must be set in root .env file")
        
        self.headers = {
            'X-API-Key': self.api_key,
            'X-API-Secret': self.api_secret,
            'Content-Type': 'application/json'
        }
    
    def get_records(self) -> List[Dict]:
        """Get all DNS records for the domain"""
        print(f"{Colors.OKBLUE}Fetching existing DNS records...{Colors.ENDC}")
        
        try:
            response = requests.get(
                f"{self.base_url}/dns/records/{self.domain}?take=100&skip=0&orderBy=type",
                headers=self.headers,
                timeout=30
            )
            response.raise_for_status()
            records = response.json()
            
            print(f"{Colors.OKGREEN}[OK] Found {len(records.get('items', []))} records{Colors.ENDC}")
            return records.get('items', [])
        
        except requests.exceptions.RequestException as e:
            print(f"{Colors.FAIL}[ERROR] Failed to fetch DNS records: {e}{Colors.ENDC}")
            raise
    
    def delete_a_record(self, record_name: str, ip_address: str) -> bool:
        """Delete an A record"""
        print(f"{Colors.OKBLUE}Deleting old A record for {record_name}.{self.domain}...{Colors.ENDC}")
        
        payload = [
            {
                "type": "A",
                "address": ip_address,
                "name": record_name
            }
        ]
        
        try:
            response = requests.request(
                method='DELETE',
                url=f"{self.base_url}/dns/records/{self.domain}",
                headers=self.headers,
                json=payload,
                timeout=30
            )
            response.raise_for_status()
            print(f"{Colors.OKGREEN}[OK] Old A record deleted{Colors.ENDC}")
            return True
        
        except requests.exceptions.RequestException as e:
            print(f"{Colors.WARNING}Warning: Failed to delete record (may not exist): {e}{Colors.ENDC}")
            return False
    
    def create_or_update_a_record(self, record_name: str, ip_address: str, ttl: int = 1800) -> bool:
        """Create or update an A record"""
        print(f"{Colors.OKBLUE}Creating A record for {record_name}.{self.domain} → {ip_address}...{Colors.ENDC}")
        
        payload = {
            "force": False,
            "items": [
                {
                    "address": ip_address,
                    "name": record_name,
                    "type": "A",
                    "ttl": ttl
                }
            ]
        }
        
        try:
            response = requests.put(
                f"{self.base_url}/dns/records/{self.domain}",
                headers=self.headers,
                json=payload,
                timeout=30
            )
            response.raise_for_status()
            print(f"{Colors.OKGREEN}[OK] DNS record updated successfully{Colors.ENDC}")
            print(f"  {record_name}.{self.domain} → {ip_address}")
            return True
        
        except requests.exceptions.RequestException as e:
            print(f"{Colors.FAIL}[ERROR] Failed to update DNS record: {e}{Colors.ENDC}")
            raise
    
    def update_api_record(self, new_ip: str, record_name: str) -> bool:
        """Update api.follow.email A record"""
        # First, try to get existing record to delete
        try:
            records = self.get_records()
            api_records = [r for r in records if r.get('name') == record_name and r.get('type') == 'A']
            
            if api_records:
                old_ip = api_records[0].get('address')
                print(f"  Old IP: {old_ip}")
                self.delete_a_record(record_name, old_ip)
                time.sleep(2)  # Wait for deletion to propagate
        except Exception as e:
            print(f"{Colors.WARNING}Note: Could not delete old record: {e}{Colors.ENDC}")
        
        # Create new record
        return self.create_or_update_a_record(record_name, new_ip)


class InstanceProvisioner:
    """Handles instance provisioning and setup"""
    
    def __init__(self, ip_address: str, ssh_key_path: str, ssh_user: str = 'ubuntu'):
        self.ip_address = ip_address
        self.ssh_key_path = ssh_key_path
        self.ssh_user = ssh_user
    
    def wait_for_ssh(self, timeout: int = 180) -> bool:
        """Wait for SSH to become available"""
        print(f"{Colors.OKBLUE}Waiting for SSH to become available...{Colors.ENDC}")
        
        start_time = time.time()
        while time.time() - start_time < timeout:
            try:
                result = subprocess.run(
                    [
                        'ssh',
                        '-o', 'StrictHostKeyChecking=no',
                        '-o', 'UserKnownHostsFile=/dev/null',
                        '-o', 'ConnectTimeout=5',
                        '-i', self.ssh_key_path,
                        f'{self.ssh_user}@{self.ip_address}',
                        'echo "SSH Ready"'
                    ],
                    capture_output=True,
                    timeout=10
                )
                
                if result.returncode == 0:
                    print(f"{Colors.OKGREEN}[OK] SSH is ready!{Colors.ENDC}")
                    return True
            except (subprocess.TimeoutExpired, Exception):
                pass
            
            print("  Waiting for SSH...")
            time.sleep(10)
        
        print(f"{Colors.FAIL}[ERROR] SSH timeout{Colors.ENDC}")
        return False
    
    def run_ssh_command(self, command: str, show_output: bool = True, timeout: int = 300) -> bool:
        """Run a command via SSH"""
        try:
            result = subprocess.run(
                [
                    'ssh',
                    '-o', 'StrictHostKeyChecking=no',
                    '-o', 'UserKnownHostsFile=/dev/null',
                    '-i', self.ssh_key_path,
                    f'{self.ssh_user}@{self.ip_address}',
                    command
                ],
                capture_output=True,
                text=True,
                timeout=timeout
            )
            
            if show_output and result.stdout:
                print(result.stdout)
            
            if result.returncode != 0:
                print(f"{Colors.FAIL}Command failed: {result.stderr}{Colors.ENDC}")
                return False
            
            return True
        
        except subprocess.TimeoutExpired:
            print(f"{Colors.FAIL}SSH command timed out after {timeout}s{Colors.ENDC}")
            return False
        except Exception as e:
            print(f"{Colors.FAIL}SSH command failed: {e}{Colors.ENDC}")
            return False
    
    def install_dependencies(self) -> bool:
        """Install Docker, nginx, and other dependencies"""
        print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
        print(f"{Colors.HEADER}Installing Dependencies{Colors.ENDC}")
        print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
        
        commands = [
            # Update system
            ("Updating system packages", "sudo apt-get update -y"),
            
            # Install prerequisites
            ("Installing prerequisites", 
             "sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common gnupg lsb-release"),
            
            # Install Docker
            ("Adding Docker GPG key",
             "curl -fsSL --max-time 30 https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg"),
            
            ("Adding Docker repository",
             "sudo bash -c 'echo \"deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable\" > /etc/apt/sources.list.d/docker.list'"),
            
            ("Updating package index",
             "sudo apt-get update -y"),
            
            ("Installing Docker",
             "sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin"),
            
            ("Starting Docker service",
             "sudo systemctl start docker && sudo systemctl enable docker"),
            
            # Install nginx
            ("Installing nginx",
             "sudo apt-get install -y nginx"),
            
            ("Enabling nginx",
             "sudo systemctl enable nginx"),
            
            # Install other utilities
            ("Installing utilities",
             "sudo apt-get install -y git curl wget htop vim"),
        ]
        
        for description, command in commands:
            print(f"{Colors.OKBLUE}{description}...{Colors.ENDC}")
            # Use appropriate timeouts - shorter for quick operations, longer for package installs
            if 'gpg' in description.lower() or 'repository' in description.lower():
                timeout = 30  # Quick operations
                show_cmd_output = True  # Show output for debugging
            elif 'updating' in description.lower() or 'installing' in description.lower():
                timeout = 600  # Package operations can take time
                show_cmd_output = False
            else:
                timeout = 300  # Default
                show_cmd_output = False
            if not self.run_ssh_command(command, show_output=show_cmd_output, timeout=timeout):
                print(f"{Colors.FAIL}[ERROR] Failed: {description}{Colors.ENDC}")
                return False
            print(f"{Colors.OKGREEN}[OK] {description} completed{Colors.ENDC}")
        
        print(f"\n{Colors.OKGREEN}[OK] All dependencies installed successfully!{Colors.ENDC}\n")
        return True
    
    def setup_nginx(self, record_name: str) -> bool:
        """Setup nginx reverse proxy"""
        print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
        print(f"{Colors.HEADER}Configuring Nginx{Colors.ENDC}")
        print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
        
        # Create nginx config content
        nginx_config = f"""server {{
            listen 80;
            server_name {record_name}.follow.email;

            # Security headers
            add_header X-Frame-Options "SAMEORIGIN" always;
            add_header X-Content-Type-Options "nosniff" always;
            add_header X-XSS-Protection "1; mode=block" always;

            # Proxy settings
            location / {{
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
            }}

            # Health check endpoint
            location /api/v1/health {{
                proxy_pass http://localhost:8080;
                access_log off;
            }}
        }}"""
        
        # Write nginx config to a temporary local file, then copy via SCP
        import tempfile
        with tempfile.NamedTemporaryFile(mode='w', suffix='.conf', delete=False) as tmp_file:
            tmp_file.write(nginx_config)
            tmp_config_path = tmp_file.name
        
        try:
            # Copy config file to server
            print(f"{Colors.OKBLUE}Copying nginx config to server...{Colors.ENDC}")
            scp_command = [
                'scp',
                '-o', 'StrictHostKeyChecking=no',
                '-o', 'UserKnownHostsFile=/dev/null',
                '-i', self.ssh_key_path,
                tmp_config_path,
                f'{self.ssh_user}@{self.ip_address}:/tmp/nginx-config.tmp'
            ]
            result = subprocess.run(scp_command, capture_output=True, timeout=30)
            if result.returncode != 0:
                print(f"{Colors.FAIL}Failed to copy nginx config: {result.stderr.decode() if result.stderr else 'Unknown error'}{Colors.ENDC}")
                if result.stdout:
                    print(f"  Output: {result.stdout.decode()}")
                return False
            print(f"{Colors.OKGREEN}[OK] Nginx config file copied to server{Colors.ENDC}")
            
            # Move file to final location immediately after SCP
            print(f"{Colors.OKBLUE}Moving nginx config to final location...{Colors.ENDC}")
            move_command = "sudo mv -f /tmp/nginx-config.tmp /etc/nginx/sites-available/follow-email && sudo chmod 644 /etc/nginx/sites-available/follow-email"
            if not self.run_ssh_command(move_command, show_output=True, timeout=10):
                print(f"{Colors.FAIL}Failed to move nginx config file{Colors.ENDC}")
                return False
            print(f"{Colors.OKGREEN}[OK] Nginx config file moved to final location{Colors.ENDC}")
            
            # Config file is already created, so skip it in commands list
            config_command = "echo 'Config already created'"
        finally:
            # Clean up local temp file
            try:
                os.unlink(tmp_config_path)
            except:
                pass
        
        commands = [
            ("Creating nginx config", config_command),
            ("Creating symbolic link", 
             "sudo ln -sf /etc/nginx/sites-available/follow-email /etc/nginx/sites-enabled/"),
            ("Removing default site",
             "[ -f /etc/nginx/sites-enabled/default ] && sudo rm -f /etc/nginx/sites-enabled/default || true"),
            ("Testing nginx config",
             "sudo nginx -t"),
            ("Restarting nginx",
             "sudo systemctl stop nginx 2>/dev/null; sudo systemctl start nginx && sleep 2 && sudo systemctl is-active --quiet nginx"),
        ]
        
        for description, command in commands:
            print(f"{Colors.OKBLUE}{description}...{Colors.ENDC}")
            # Use appropriate timeouts - longer for service operations
            if 'Removing' in description or 'Testing' in description:
                timeout = 10
            elif 'Restarting' in description:
                timeout = 60  # Service operations can take time
            elif 'Creating nginx config' in description:
                timeout = 10  # File move should be instant
                show_cmd_output = True  # Show output to debug
            else:
                timeout = 30
            if not self.run_ssh_command(command, show_output=True, timeout=timeout):
                # For removing default site, it's OK if it doesn't exist
                if 'Removing default site' in description:
                    print(f"{Colors.WARNING}Note: Default site may not exist, continuing...{Colors.ENDC}")
                # For restarting nginx, check if it's actually running
                elif 'Restarting nginx' in description:
                    print(f"{Colors.WARNING}Restart command failed, checking if nginx is running...{Colors.ENDC}")
                    check_cmd = "sudo systemctl is-active --quiet nginx && echo 'nginx is running' || echo 'nginx is not running'"
                    if self.run_ssh_command(check_cmd, show_output=True, timeout=10):
                        print(f"{Colors.OKGREEN}Nginx is running, continuing...{Colors.ENDC}")
                    else:
                        print(f"{Colors.FAIL}[ERROR] Nginx is not running after restart attempt{Colors.ENDC}")
                        return False
                else:
                    print(f"{Colors.FAIL}[ERROR] Failed: {description}{Colors.ENDC}")
                    return False
            print(f"{Colors.OKGREEN}[OK] {description} completed{Colors.ENDC}")
        
        # Verify nginx is running and listening
        print(f"\n{Colors.OKBLUE}Verifying nginx status...{Colors.ENDC}")
        nginx_status = "sudo systemctl is-active nginx && echo 'Nginx is active' || echo 'Nginx is not active'"
        self.run_ssh_command(nginx_status, show_output=True, timeout=10)
        
        port_check = "sudo netstat -tlnp | grep ':80 ' || sudo ss -tlnp | grep ':80 ' || echo 'Port 80 check failed'"
        self.run_ssh_command(port_check, show_output=True, timeout=10)
        
        print(f"\n{Colors.OKGREEN}[OK] Nginx configured successfully!{Colors.ENDC}\n")
        print(f"{Colors.WARNING}IMPORTANT: Make sure your Excloud security group allows HTTP (port 80) traffic!{Colors.ENDC}")
        print(f"{Colors.WARNING}Security Group ID: {os.getenv('EXCLOUD_SECURITY_GROUP_ID', 'Check your Excloud console')}{Colors.ENDC}\n")
        return True
    
    def setup_env_file(self, local_env_path: str = None) -> bool:
        """
        Copy .env file to server and set up environment variables
        
        This copies the .env file and creates a script to export variables
        so the application can access them.
        """
        print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
        print(f"{Colors.HEADER}Setting Up Environment Variables{Colors.ENDC}")
        print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
        
        # Default to root .env file if not specified
        if local_env_path is None:
            local_env_path = os.path.join(os.path.dirname(__file__), '../../../.env')
        
        # First check if .env exists locally
        if not os.path.exists(local_env_path):
            print(f"{Colors.WARNING}.env file not found at {local_env_path}{Colors.ENDC}")
            print(f"{Colors.WARNING}Skipping env setup. You'll need to copy it manually.{Colors.ENDC}")
            return False
        
        print(f"{Colors.OKBLUE}Copying .env file to server...{Colors.ENDC}")
        
        # Use scp to copy the .env file to root of the app directory AND infra directory
        scp_commands = [
            (f'{self.ssh_user}@{self.ip_address}:/opt/follow.email/.env', 'Root .env file'),
            (f'{self.ssh_user}@{self.ip_address}:/opt/follow.email/infra/.env', 'Infra .env file (for docker-compose)'),
        ]
        
        for remote_path, description in scp_commands:
            scp_command = [
                'scp',
                '-o', 'StrictHostKeyChecking=no',
                '-o', 'UserKnownHostsFile=/dev/null',
                '-i', self.ssh_key_path,
                local_env_path,
                remote_path
            ]
            
            try:
                result = subprocess.run(scp_command, capture_output=True, timeout=60)
                if result.returncode != 0:
                    print(f"{Colors.WARNING}Failed to copy .env to {description}{Colors.ENDC}")
                    print(f"  Error: {result.stderr.decode() if result.stderr else 'Unknown error'}")
                else:
                    print(f"{Colors.OKGREEN}[OK] {description} copied successfully{Colors.ENDC}")
            except Exception as e:
                print(f"{Colors.WARNING}Failed to copy .env to {description}: {e}{Colors.ENDC}")
        
        print(f"\n{Colors.OKGREEN}[OK] Environment variables configured!{Colors.ENDC}\n")
        return True
    
    def deploy_application(self, github_repo: Optional[str] = None) -> bool:
        """Deploy the application"""
        print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
        print(f"{Colors.HEADER}Deploying Application{Colors.ENDC}")
        print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
        
        if not github_repo:
            github_repo = os.getenv('GITHUB_REPO', 'https://github.com/followdotemail/follow.email.git')
        
        commands = [
            ("Creating app directory",
             "sudo mkdir -p /opt/follow.email && sudo chown $USER:$USER /opt/follow.email"),
            
            ("Cloning repository",
             f"git clone {github_repo} /opt/follow.email || (cd /opt/follow.email && git pull)"),
        ]
        
        for description, command in commands:
            print(f"{Colors.OKBLUE}{description}...{Colors.ENDC}")
            if not self.run_ssh_command(command, show_output=True):
                print(f"{Colors.WARNING}Note: {description} - check if already exists{Colors.ENDC}")
        
        print(f"\n{Colors.OKGREEN}[OK] Application deployment base setup complete!{Colors.ENDC}")
        
        return True
    
    def start_application(self) -> bool:
        """Start the application using docker-compose"""
        print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
        print(f"{Colors.HEADER}Starting Application{Colors.ENDC}")
        print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
        
        commands = [
            ("Building Docker image",
             "cd /opt/follow.email/infra && sudo docker compose build backend"),
            
            ("Starting application",
             "cd /opt/follow.email/infra && sudo docker compose up -d backend"),
            
            ("Waiting for app to be healthy",
             "sleep 10"),
        ]
        
        for description, command in commands:
            print(f"{Colors.OKBLUE}{description}...{Colors.ENDC}")
            if not self.run_ssh_command(command, show_output=True, timeout=600):
                print(f"{Colors.FAIL}[ERROR] Failed: {description}{Colors.ENDC}")
                return False
            print(f"{Colors.OKGREEN}[OK] {description} completed{Colors.ENDC}")
        
        # Check container status
        print(f"\n{Colors.OKBLUE}Checking container status...{Colors.ENDC}")
        status_check = "cd /opt/follow.email/infra && sudo docker compose ps backend"
        self.run_ssh_command(status_check, show_output=True, timeout=10)
        
        # Check if the app is responding
        print(f"\n{Colors.OKBLUE}Checking application health...{Colors.ENDC}")
        health_check = "curl -f http://localhost:8080/api/v1/health || echo 'Health check failed'"
        if self.run_ssh_command(health_check, show_output=True, timeout=10):
            print(f"{Colors.OKGREEN}[OK] Application is running and healthy!{Colors.ENDC}")
        else:
            print(f"{Colors.WARNING}Warning: Health check failed{Colors.ENDC}")
            print(f"{Colors.OKBLUE}Checking container logs...{Colors.ENDC}")
            logs_check = "cd /opt/follow.email/infra && sudo docker compose logs --tail=50 backend"
            self.run_ssh_command(logs_check, show_output=True, timeout=30)
            print(f"{Colors.WARNING}Please check the logs above for errors{Colors.ENDC}")
        
        print(f"\n{Colors.OKGREEN}[OK] Application started successfully!{Colors.ENDC}\n")
        return True


def main():
    parser = argparse.ArgumentParser(description='Provision Excloud development instance for follow.email')
    parser.add_argument(
        '--action',
        choices=['create', 'destroy', 'setup-only'],
        required=True,
        help='Action to perform'
    )
    parser.add_argument(
        '--instance-id',
        help='VM ID (required for destroy action) - can also use --vm-id'
    )
    parser.add_argument(
        '--vm-id',
        dest='instance_id',
        help='VM ID (alias for --instance-id)'
    )
    parser.add_argument(
        '--ssh-key',
        default=os.path.expanduser('~/.ssh/id_ed25519'),
        help='Path to SSH private key'
    )
    parser.add_argument(
        '--github-repo',
        default='https://github.com/followdotemail/follow.email.git',
        help='GitHub repository URL'
    )
    parser.add_argument(
        '--record-name',
        default='api',
        help='DNS record name (e.g., api for api.follow.email)'
    )
    parser.add_argument(
        '--troubleshoot',
        action='store_true',
        help='Run troubleshooting diagnostics on the server'
    )
    
    args = parser.parse_args()
    
    # Handle troubleshoot action
    if args.troubleshoot:
        if not args.instance_id:
            print(f"{Colors.FAIL}--instance-id (or --vm-id) required for troubleshoot action{Colors.ENDC}")
            sys.exit(1)
        
        # Read instance info
        try:
            with open('instance-info.json', 'r') as f:
                instance = json.load(f)
            instance_ip = instance.get('public_ipv4')
        except FileNotFoundError:
            print(f"{Colors.FAIL}instance-info.json not found. Please provide instance IP with --instance-id{Colors.ENDC}")
            sys.exit(1)
        
        provisioner = InstanceProvisioner(instance_ip, args.ssh_key)
        
        print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
        print(f"{Colors.HEADER}Troubleshooting Backend Deployment{Colors.ENDC}")
        print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
        
        # Check .env file
        print(f"{Colors.OKBLUE}1. Checking .env file...{Colors.ENDC}")
        provisioner.run_ssh_command("ls -la /opt/follow.email/infra/.env && echo '---' && head -5 /opt/follow.email/infra/.env | grep -v 'PASSWORD\\|SECRET\\|KEY'", show_output=True)
        
        # Check container status
        print(f"\n{Colors.OKBLUE}2. Checking container status...{Colors.ENDC}")
        provisioner.run_ssh_command("cd /opt/follow.email/infra && sudo docker compose ps", show_output=True)
        
        # Check container logs
        print(f"\n{Colors.OKBLUE}3. Checking backend container logs (last 50 lines)...{Colors.ENDC}")
        provisioner.run_ssh_command("cd /opt/follow.email/infra && sudo docker compose logs --tail=50 backend", show_output=True)
        
        # Check port 8080
        print(f"\n{Colors.OKBLUE}4. Checking if port 8080 is listening...{Colors.ENDC}")
        provisioner.run_ssh_command("sudo netstat -tlnp | grep :8080 || sudo ss -tlnp | grep :8080 || echo 'Port 8080 is not listening'", show_output=True)
        
        # Test backend health
        print(f"\n{Colors.OKBLUE}5. Testing backend health endpoint...{Colors.ENDC}")
        provisioner.run_ssh_command("curl -v http://localhost:8080/api/v1/health 2>&1 | head -20", show_output=True)
        
        # Check nginx error logs
        print(f"\n{Colors.OKBLUE}6. Checking nginx error logs (last 20 lines)...{Colors.ENDC}")
        provisioner.run_ssh_command("sudo tail -20 /var/log/nginx/error.log", show_output=True)
        
        print(f"\n{Colors.OKGREEN}Troubleshooting complete!{Colors.ENDC}\n")
        sys.exit(0)
    
    try:
        if args.action == 'create':
            print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
            print(f"{Colors.HEADER}Follow.Email Backend Instance Provisioning{Colors.ENDC}")
            print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
            
            # Step 1: Create instance
            excloud = ExcloudInstanceManager()
            
            # Debug: Print environment variables
            print(f"\n{Colors.HEADER}Environment Variables Loaded:{Colors.ENDC}")
            print(f"  EXCLOUD_IMAGE_ID: {os.getenv('EXCLOUD_IMAGE_ID', 'NOT SET')}")
            print(f"  EXCLOUD_ZONE_ID: {os.getenv('EXCLOUD_ZONE_ID', 'NOT SET')}")
            print(f"  EXCLOUD_SUBNET_ID: {os.getenv('EXCLOUD_SUBNET_ID', 'NOT SET')}")
            print(f"  EXCLOUD_PROJECT_ID: {os.getenv('EXCLOUD_PROJECT_ID', 'NOT SET')}")
            print(f"  EXCLOUD_SSH_PUBKEY: {os.getenv('EXCLOUD_SSH_PUBKEY', 'NOT SET')}\n")
            
            # Note: You need to provide these IDs from your Excloud account
            # Get them from the Excloud console or API
            zone_id = int(os.getenv('EXCLOUD_ZONE_ID', 1))
            
            instance_config = {
                "name": "follow-email-backend",
                "instance_type": "t1.medium",  # 2 vCPU, 8 GiB Memory
                "image_id": int(os.getenv('EXCLOUD_IMAGE_ID', 10)),  # Ubuntu 22.04 image ID
                "zone_id": zone_id,
                "subnet_id": int(os.getenv('EXCLOUD_SUBNET_ID', 1273)),  # Subnet ID
                "allocate_public_ipv4": True,
                "ssh_pubkey": os.getenv('EXCLOUD_SSH_PUBKEY'),  # SSH public key content
                "security_group_ids": [int(os.getenv('EXCLOUD_SECURITY_GROUP_ID', 1624))],
                "project_id": int(os.getenv('EXCLOUD_PROJECT_ID', 1)) if os.getenv('EXCLOUD_PROJECT_ID') else int(1),
                "root_volume": {
                    "size_gib": 24,  # Root volume size
                    "zone_id": zone_id,
                    "baseline_iops": 3000,  # Baseline IOPS (3000-16000 allowed)
                    "baseline_throughput_mbps": 250  # Baseline Throughput MB/s
                }
            }
            
            instance = excloud.create_instance(instance_config)
            vm_id = instance.get('vm_id')
            instance_ip = instance.get('public_ipv4')
            zone_id = instance_config.get('zone_id')
            
            # Save instance info
            with open('instance-info.json', 'w') as f:
                json.dump(instance, f, indent=2)
            print(f"\n{Colors.OKGREEN}Instance info saved to instance-info.json{Colors.ENDC}\n")
            
            # Step 2: Wait for instance to be ready
            if not excloud.wait_for_instance_ready(vm_id, zone_id):
                print(f"{Colors.FAIL}Instance failed to start{Colors.ENDC}")
                sys.exit(1)
            
            # Step 3: Update DNS
            dns = SpaceshipDNSManager()
            record_name = args.record_name
            if not dns.update_api_record(instance_ip, record_name):
                print(f"{Colors.FAIL}Failed to update DNS{Colors.ENDC}")
                sys.exit(1)
            
            # Step 4: Setup instance
            provisioner = InstanceProvisioner(instance_ip, args.ssh_key)
            
            if not provisioner.wait_for_ssh():
                print(f"{Colors.FAIL}SSH not available{Colors.ENDC}")
                sys.exit(1)
            
            if not provisioner.install_dependencies():
                print(f"{Colors.FAIL}Failed to install dependencies{Colors.ENDC}")
                sys.exit(1)
            
            if not provisioner.setup_nginx(record_name):
                print(f"{Colors.FAIL}Failed to setup nginx{Colors.ENDC}")
                sys.exit(1)
            
            if not provisioner.deploy_application(args.github_repo):
                print(f"{Colors.FAIL}Failed to deploy application{Colors.ENDC}")
                sys.exit(1)
            
            # Step 5: Setup environment variables
            print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
            print(f"{Colors.HEADER}Step 5: Setting Up Environment{Colors.ENDC}")
            print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
            
            provisioner.setup_env_file()
            
            # Step 6: Start the application
            if not provisioner.start_application():
                print(f"{Colors.WARNING}Warning: Failed to start application automatically{Colors.ENDC}")
                print(f"{Colors.WARNING}You can start it manually after provisioning{Colors.ENDC}")
            
            # Success!
            print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
            print(f"{Colors.OKGREEN}{'[OK] PROVISIONING COMPLETE!':^60}{Colors.ENDC}")
            print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
            
            print(f"{Colors.BOLD}Instance Details:{Colors.ENDC}")
            print(f"  VM ID:       {vm_id}")
            print(f"  IP Address:  {instance_ip}")
            print(f"  SSH Access:  ssh -i {args.ssh_key} ubuntu@{instance_ip}")
            print(f"  API URL:     http://{record_name}.follow.email")
            print(f"\n{Colors.BOLD}Quick Commands:{Colors.ENDC}")
            print(f"  Check status: ssh -i {args.ssh_key} ubuntu@{instance_ip} 'cd /opt/follow.email/infra && sudo docker compose ps'")
            print(f"  View logs:    ssh -i {args.ssh_key} ubuntu@{instance_ip} 'cd /opt/follow.email/infra && sudo docker compose logs -f backend'")
            print(f"  Restart app:  ssh -i {args.ssh_key} ubuntu@{instance_ip} 'cd /opt/follow.email/infra && sudo docker compose restart backend'")
            print(f"  Test API:     curl http://{record_name}.follow.email/api/v1/health")
            print(f"\n{Colors.OKGREEN}[OK] Follow.Email Backend is now running on {record_name}.follow.email!{Colors.ENDC}\n")
        
        elif args.action == 'destroy':
            if not args.instance_id:
                print(f"{Colors.FAIL}--instance-id (or --vm-id) required for destroy action{Colors.ENDC}")
                sys.exit(1)
            
            # Try to convert to int (vm_id)
            try:
                vm_id = int(args.instance_id)
            except ValueError:
                print(f"{Colors.FAIL}VM ID must be an integer{Colors.ENDC}")
                sys.exit(1)
            
            excloud = ExcloudInstanceManager()
            excloud.delete_instance(vm_id)
        
        elif args.action == 'setup-only':
            # Read instance info
            with open('instance-info.json', 'r') as f:
                instance = json.load(f)
            
            instance_ip = instance.get('public_ipv4')
            record_name = args.record_name
            provisioner = InstanceProvisioner(instance_ip, args.ssh_key)
            
            if not provisioner.wait_for_ssh():
                sys.exit(1)
            
            provisioner.install_dependencies()
            provisioner.setup_nginx(record_name)
            provisioner.deploy_application(args.github_repo)
            provisioner.setup_env_file()
            provisioner.start_application()
            
            print(f"\n{Colors.OKGREEN}[OK] Setup complete! Application running at http://{record_name}.follow.email{Colors.ENDC}\n")
    
    except KeyboardInterrupt:
        print(f"\n{Colors.WARNING}Interrupted by user{Colors.ENDC}")
        sys.exit(1)
    except Exception as e:
        print(f"\n{Colors.FAIL}Error: {e}{Colors.ENDC}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
