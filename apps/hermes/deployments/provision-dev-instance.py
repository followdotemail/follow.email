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
    cd apps/hermes/deployments
    python provision-dev-instance.py --action create --record-name api
    python provision-dev-instance.py --action destroy --vm-id 1234
    python provision-dev-instance.py --action setup-only
"""

"""
#######################################
NEED TO MAKE THE SCRIPT FAILED PROOF -
- Need to separate the cli command execution in 2 parts.
- 1st part will retry for some time with exponential backoff. This part will have critical command which should be retried.
- 2nd part will have try catch block only with no retry. This part will have non-critical commands which can be ignored if they fail.
- First, I need to the command into critical and non-critical parts.
#######################################
"""

from dis import show_code
import os
import sys
import time
import json
import argparse
import subprocess
from datetime import datetime
from typing import Dict, Optional, List, Tuple
from dataclasses import dataclass
from enum import Enum
import requests
from dotenv import load_dotenv


class CommandCriticality(Enum):
    """Classification of command criticality for error handling"""
    CRITICAL = "critical"           # Must succeed, retry with backoff
    IMPORTANT = "important"         # Should succeed, retry once, warn on failure
    OPTIONAL = "optional"           # Nice to have, no retry, ignore on failure


@dataclass
class CommandConfig:
    """Configuration for command execution"""
    description: str
    command: str
    criticality: CommandCriticality = CommandCriticality.IMPORTANT
    timeout: int = 300              # Default timeout in seconds
    max_retries: int = 3            # Maximum retry attempts for critical commands
    initial_delay: float = 5.0      # Initial delay before retry (seconds)
    max_delay: float = 60.0         # Maximum delay between retries
    show_output: bool = False       # Whether to show command output
    check_running: bool = False     # Check if command is still running after timeout


@dataclass 
class CommandResult:
    """Result of command execution"""
    success: bool
    stdout: str = ""
    stderr: str = ""
    return_code: int = -1
    timed_out: bool = False
    still_running: bool = False
    attempts: int = 1

# Load environment variables from root .env file
env_path = os.path.join(os.path.dirname(__file__), '../../../.env')
load_dotenv(env_path)
print(f"Loading environment from: {os.path.abspath(env_path)}")
webhook_url = "https://discordapp.com/api/webhooks/1440730059056091147/b7hiFgSWYS3kb1a6GUt86CQFRLHLjWSV5YegU1BEw9PaBlAKRtpZViHX5L9aF79vLAfp"

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


class DiscordNotifier:
    """Sends notifications to a Discord channel"""

    def __init__(self) -> None:
        self.webhook_url = webhook_url or os.getenv('DISCORD_WEBHOOK_URL')

    def send_notification(self, title: str, description: str, color: int = 0x00ff00, fields: List[Dict] = None):
        """send an embedded message to Discord"""
        if not self.webhook_url:
            return

        payload = {
            "embeds": [
                {
                    "title": title,
                    "description": description,
                    "color": color,
                    "fields": fields or [],
                    "footer": {
                        "text": f"Follow.Email Provisioning • {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
                    }
                }
            ]
        }

        try:
            requests.post(self.webhook_url, json=payload, timeout=100)
        except Exception as e:
            print(f"{Colors.FAIL}[ERROR] Failed to send notification: {e}{Colors.ENDC}")

    def success(self, title: str, description: str, fields: List[Dict] = None):
        self.send_notification(title=title, description=description, color=0x00ff00, fields=fields)

    def error(self, title: str, description: str, fields: List[Dict] = None):
        self.send_notification(title=title, description=description, color=0xff0000, fields=fields)

    def info(self, title: str, description: str, fields: List[Dict] = None):
        self.send_notification(title=title, description=description, color=0x0000ff, fields=fields)

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
                timeout=100
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
                timeout=100
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
                timeout=100
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
                timeout=100
            )

            if response.status_code >= 200 and response.status_code <= 299:
                records = response.json()
                print(f"{Colors.OKGREEN}[OK] Found {len(records.get('items', []))} records{Colors.ENDC}")
                return records.get('items', [])
            else:
                print(f"{Colors.FAIL}[ERROR] Failed to fetch DNS records: {response.status_code}{Colors.ENDC}")
                print(f"{Colors.FAIL}[ERROR] Response body: {response.text}{Colors.ENDC}")
                raise requests.exceptions.RequestException(f"Failed to fetch DNS records: {response.status_code}")
            
        
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
                timeout=300
            )
            response.raise_for_status()
            print(f"{Colors.OKGREEN}[OK] Old A record deleted{Colors.ENDC}")
            return True
        
        except requests.exceptions.RequestException as e:
            print(f"{Colors.WARNING}Warning: Failed to delete record (may not exist): {e}{Colors.ENDC}")
            return False
    
    def create_or_update_a_record(self, record_name: str, ip_address: str, ttl: int = 1800) -> bool:
        """Create or update an A record"""
        print(f"{Colors.OKBLUE}Creating A record for {record_name}.{self.domain} -> {ip_address}...{Colors.ENDC}")
        
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
                timeout=300
            )
            response.raise_for_status()
            print(f"{Colors.OKGREEN}[OK] DNS record updated successfully{Colors.ENDC}")
            print(f"  {record_name}.{self.domain} -> {ip_address}")
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
                    timeout=100
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
        """Run a command via SSH (simple wrapper for backward compatibility)"""
        result = self._execute_ssh_command(command, show_output, timeout)
        return result.success

    def _execute_ssh_command(self, command: str, show_output: bool = True, timeout: int = 300) -> CommandResult:
        """Execute SSH command and return detailed result"""
        try:
            result = subprocess.run(
                [
                    'ssh',
                    '-o', 'StrictHostKeyChecking=no',
                    '-o', 'UserKnownHostsFile=/dev/null',
                    '-o', 'ServerAliveInterval=30',
                    '-o', 'ServerAliveCountMax=3',
                    '-i', self.ssh_key_path,
                    f'{self.ssh_user}@{self.ip_address}',
                    command
                ],
                capture_output=True,
                text=True,
                encoding='utf-8',
                errors='replace',
                timeout=timeout
            )
            
            if show_output:
                if result.stdout:
                    print(result.stdout)
                if result.stderr:
                    print(result.stderr)
            
            if result.returncode != 0:
                if not show_output and result.stderr:
                    print(f"{Colors.FAIL}Command failed: {result.stderr}{Colors.ENDC}")
                else:
                    print(f"{Colors.FAIL}Command failed with return code {result.returncode}{Colors.ENDC}")
                return CommandResult(
                    success=False,
                    stdout=result.stdout,
                    stderr=result.stderr,
                    return_code=result.returncode
                )
            
            return CommandResult(
                success=True,
                stdout=result.stdout,
                stderr=result.stderr,
                return_code=result.returncode
            )
        
        except subprocess.TimeoutExpired:
            print(f"{Colors.FAIL}SSH command timed out after {timeout}s{Colors.ENDC}")
            return CommandResult(success=False, timed_out=True)
        except Exception as e:
            print(f"{Colors.FAIL}SSH command failed: {e}{Colors.ENDC}")
            return CommandResult(success=False, stderr=str(e))

    def _check_command_still_running(self, command_pattern: str) -> bool:
        """Check if a command matching the pattern is still running on the server"""
        # Use pgrep to check if process is running
        check_cmd = f"pgrep -f '{command_pattern}' > /dev/null 2>&1 && echo 'RUNNING' || echo 'NOT_RUNNING'"
        result = self._execute_ssh_command(check_cmd, show_output=False, timeout=30)
        return result.success and 'RUNNING' in result.stdout

    def _wait_for_command_completion(self, command_pattern: str, timeout: int = 600) -> bool:
        """Wait for a command to complete that might still be running after timeout"""
        print(f"{Colors.OKBLUE}Waiting for command to complete...{Colors.ENDC}")
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            if not self._check_command_still_running(command_pattern):
                print(f"{Colors.OKGREEN}Command completed.{Colors.ENDC}")
                return True
            print(f"  Still running... ({int(time.time() - start_time)}s elapsed)")
            time.sleep(10)
        
        print(f"{Colors.WARNING}Command still running after {timeout}s wait{Colors.ENDC}")
        return False

    def run_command_with_retry(self, config: CommandConfig) -> CommandResult:
        """
        Execute a command with retry logic based on criticality.
        
        For CRITICAL commands: Retry with exponential backoff
        For IMPORTANT commands: Retry once, then warn
        For OPTIONAL commands: No retry, ignore failure
        """
        attempt = 0
        delay = config.initial_delay
        last_result = CommandResult(success=False)
        
        # Determine max retries based on criticality
        if config.criticality == CommandCriticality.CRITICAL:
            max_attempts = config.max_retries
        elif config.criticality == CommandCriticality.IMPORTANT:
            max_attempts = 2
        else:  # OPTIONAL
            max_attempts = 1
        
        while attempt < max_attempts:
            attempt += 1
            
            if attempt > 1:
                print(f"{Colors.WARNING}Retry attempt {attempt}/{max_attempts} after {delay:.1f}s delay...{Colors.ENDC}")
                time.sleep(delay)
                # Exponential backoff for next attempt
                delay = min(delay * 2, config.max_delay)
            
            last_result = self._execute_ssh_command(
                config.command,
                show_output=config.show_output,
                timeout=config.timeout
            )
            last_result.attempts = attempt
            
            if last_result.success:
                return last_result
            
            # Handle timeout - check if command is still running
            if last_result.timed_out and config.check_running:
                # Extract a pattern from the command to check
                cmd_pattern = config.command.split()[0] if config.command else ""
                if cmd_pattern in ['apt-get', 'apt', 'docker', 'git']:
                    cmd_pattern = config.command.split('&&')[0].strip() if '&&' in config.command else config.command
                
                if self._check_command_still_running(cmd_pattern):
                    last_result.still_running = True
                    print(f"{Colors.WARNING}Command timed out but still running on server{Colors.ENDC}")
                    
                    # Wait for completion for critical commands
                    if config.criticality == CommandCriticality.CRITICAL:
                        if self._wait_for_command_completion(cmd_pattern, timeout=config.timeout * 2):
                            # Verify the command completed successfully
                            verify_result = self._execute_ssh_command("echo 'OK'", show_output=False, timeout=30)
                            if verify_result.success:
                                last_result.success = True
                                return last_result
            
            # Log based on criticality
            if config.criticality == CommandCriticality.OPTIONAL:
                print(f"{Colors.WARNING}Optional command failed, continuing...{Colors.ENDC}")
                break
            elif attempt >= max_attempts:
                if config.criticality == CommandCriticality.CRITICAL:
                    print(f"{Colors.FAIL}Critical command failed after {attempt} attempts{Colors.ENDC}")
                else:
                    print(f"{Colors.WARNING}Command failed after {attempt} attempts{Colors.ENDC}")
        
        return last_result

    def run_commands_batch(self, commands: List[CommandConfig]) -> Tuple[bool, List[CommandResult]]:
        """
        Run a batch of commands with proper error handling.
        Returns (overall_success, list_of_results)
        """
        results = []
        overall_success = True
        
        for config in commands:
            print(f"{Colors.OKBLUE}{config.description}...{Colors.ENDC}")
            
            result = self.run_command_with_retry(config)
            results.append(result)
            
            if result.success:
                print(f"{Colors.OKGREEN}[OK] {config.description} completed{Colors.ENDC}")
            else:
                if config.criticality == CommandCriticality.CRITICAL:
                    print(f"{Colors.FAIL}[FAILED] {config.description} - Critical failure{Colors.ENDC}")
                    overall_success = False
                    break  # Stop on critical failure
                elif config.criticality == CommandCriticality.IMPORTANT:
                    print(f"{Colors.WARNING}[WARN] {config.description} - Continuing despite failure{Colors.ENDC}")
                else:
                    print(f"{Colors.WARNING}[SKIP] {config.description} - Optional, skipped{Colors.ENDC}")
        
        return overall_success, results
    
    def install_dependencies(self) -> bool:
        """Install Docker, nginx, and other dependencies with retry logic"""
        print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
        print(f"{Colors.HEADER}Installing Dependencies{Colors.ENDC}")
        print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
        
        commands = [
            # Update system - Critical, must succeed
            CommandConfig(
                description="Updating system packages",
                command="sudo apt-get update -y",
                criticality=CommandCriticality.CRITICAL,
                timeout=600,
                max_retries=3,
                check_running=True
            ),
            
            # Install prerequisites - Critical
            CommandConfig(
                description="Installing prerequisites",
                command="sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common gnupg lsb-release",
                criticality=CommandCriticality.CRITICAL,
                timeout=600,
                max_retries=3,
                check_running=True
            ),
            
            # Docker GPG key - Important (might already exist)
            CommandConfig(
                description="Adding Docker GPG key",
                command="curl -fsSL --max-time 60 https://download.docker.com/linux/ubuntu/gpg | sudo gpg --batch --yes --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg",
                criticality=CommandCriticality.IMPORTANT,
                timeout=120,
                max_retries=3,
                show_output=True
            ),
            
            # Docker repository - Important
            CommandConfig(
                description="Adding Docker repository",
                command="sudo bash -c 'echo \"deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable\" > /etc/apt/sources.list.d/docker.list'",
                criticality=CommandCriticality.IMPORTANT,
                timeout=60,
                max_retries=2,
                show_output=True
            ),
            
            # Update package index after adding repo - Critical
            CommandConfig(
                description="Updating package index",
                command="sudo apt-get update -y",
                criticality=CommandCriticality.CRITICAL,
                timeout=600,
                max_retries=3,
                check_running=True
            ),
            
            # Install Docker - Critical
            CommandConfig(
                description="Installing Docker",
                command="sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin",
                criticality=CommandCriticality.CRITICAL,
                timeout=600,
                max_retries=3,
                check_running=True
            ),
            
            # Start Docker service - Critical
            CommandConfig(
                description="Starting Docker service",
                command="sudo systemctl start docker && sudo systemctl enable docker",
                criticality=CommandCriticality.CRITICAL,
                timeout=120,
                max_retries=3
            ),
            
            # Install nginx - Critical
            CommandConfig(
                description="Installing nginx",
                command="sudo apt-get install -y nginx",
                criticality=CommandCriticality.CRITICAL,
                timeout=600,
                max_retries=3,
                check_running=True
            ),
            
            # Enable nginx - Important
            CommandConfig(
                description="Enabling nginx",
                command="sudo systemctl enable nginx",
                criticality=CommandCriticality.IMPORTANT,
                timeout=60,
                max_retries=2
            ),
            
            # Install utilities - Optional (nice to have)
            CommandConfig(
                description="Installing utilities",
                command="sudo apt-get install -y git curl wget htop vim",
                criticality=CommandCriticality.OPTIONAL,
                timeout=300,
                max_retries=1,
                check_running=True
            ),
        ]
        
        success, results = self.run_commands_batch(commands)
        
        if success:
            print(f"\n{Colors.OKGREEN}[OK] All dependencies installed successfully!{Colors.ENDC}\n")
        else:
            # Check which critical command failed
            for i, (config, result) in enumerate(zip(commands, results)):
                if not result.success and config.criticality == CommandCriticality.CRITICAL:
                    print(f"\n{Colors.FAIL}[ERROR] Critical dependency installation failed: {config.description}{Colors.ENDC}")
                    break
        
        return success
    
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
            result = subprocess.run(scp_command, capture_output=True, text=True, timeout=600)
            
            # Show output for debugging
            if result.stdout:
                print(f"SCP stdout: {result.stdout}")
            if result.stderr:
                print(f"SCP stderr: {result.stderr}")
            
            if result.returncode != 0:
                print(f"{Colors.FAIL}Failed to copy nginx config (return code: {result.returncode}){Colors.ENDC}")
                return False
            
            # Verify the file actually exists on the server
            verify_result = subprocess.run(
                [
                    'ssh',
                    '-o', 'StrictHostKeyChecking=no',
                    '-o', 'UserKnownHostsFile=/dev/null',
                    '-i', self.ssh_key_path,
                    f'{self.ssh_user}@{self.ip_address}',
                    'ls -la /tmp/nginx-config.tmp'
                ],
                capture_output=True,
                text=True,
                timeout=300
            )
            
            if verify_result.returncode != 0:
                print(f"{Colors.FAIL}File verification failed - nginx config was not copied to /tmp/nginx-config.tmp{Colors.ENDC}")
                print(f"Verification output: {verify_result.stderr}")
                return False
            
            print(f"{Colors.OKGREEN}[OK] Nginx config file copied to server{Colors.ENDC}")
            print(f"File details: {verify_result.stdout.strip()}")
            
            # Move file to final location immediately after SCP
            print(f"{Colors.OKBLUE}Moving nginx config to final location...{Colors.ENDC}")
            move_command = "sudo mv -f /tmp/nginx-config.tmp /etc/nginx/sites-available/follow-email && sudo chmod 644 /etc/nginx/sites-available/follow-email"
            if not self.run_ssh_command(move_command, show_output=True, timeout=600):
                print(f"{Colors.FAIL}Failed to move nginx config file{Colors.ENDC}")
                return False
            print(f"{Colors.OKGREEN}[OK] Nginx config file moved to final location{Colors.ENDC}")
            
            # Config file is already created, so skip it in commands list
            # Use a simple verification command that always succeeds
            config_command = "[ -f /etc/nginx/sites-available/follow-email ] && echo 'OK' || (echo 'File missing' && exit 1)"
        finally:
            # Clean up local temp file
            try:
                os.unlink(tmp_config_path)
            except:
                pass
        
        commands = [
            # Creating symbolic link - Important
            CommandConfig(
                description="Creating symbolic link",
                command="sudo ln -sf /etc/nginx/sites-available/follow-email /etc/nginx/sites-enabled/",
                criticality=CommandCriticality.IMPORTANT,
                timeout=60,
                max_retries=2,
                show_output=True
            ),
            # Removing default site - Optional (might not exist)
            CommandConfig(
                description="Removing default site",
                command="[ -f /etc/nginx/sites-enabled/default ] && sudo rm -f /etc/nginx/sites-enabled/default || true",
                criticality=CommandCriticality.OPTIONAL,
                timeout=60,
                max_retries=1,
                show_output=True
            ),
            # Testing nginx config - Critical
            CommandConfig(
                description="Testing nginx config",
                command="sudo nginx -t < /dev/null 2>&1",
                criticality=CommandCriticality.CRITICAL,
                timeout=60,
                max_retries=2,
                show_output=True
            ),
            # Restarting nginx - Critical
            CommandConfig(
                description="Restarting nginx",
                command="sudo systemctl stop nginx 2>/dev/null; sudo systemctl start nginx && sleep 2 && sudo systemctl is-active --quiet nginx",
                criticality=CommandCriticality.CRITICAL,
                timeout=120,
                max_retries=3,
                show_output=True
            ),
        ]
        
        success, results = self.run_commands_batch(commands)
        
        if not success:
            # Check if nginx is at least running
            print(f"{Colors.WARNING}Checking if nginx is still running...{Colors.ENDC}")
            check_config = CommandConfig(
                description="Checking nginx status",
                command="sudo systemctl is-active --quiet nginx && echo 'nginx is running' || echo 'nginx is not running'",
                criticality=CommandCriticality.OPTIONAL,
                timeout=60,
                show_output=True
            )
            check_result = self.run_command_with_retry(check_config)
            if 'running' in check_result.stdout:
                print(f"{Colors.OKGREEN}Nginx is running, continuing...{Colors.ENDC}")
            else:
                print(f"{Colors.FAIL}[ERROR] Nginx is not running{Colors.ENDC}")
                return False
        
        # Verify nginx is running and listening
        print(f"\n{Colors.OKBLUE}Verifying nginx status...{Colors.ENDC}")
        nginx_status = "sudo systemctl is-active nginx && echo 'Nginx is active' || echo 'Nginx is not active'"
        self.run_ssh_command(nginx_status, show_output=True, timeout=100)
        
        port_check = "sudo netstat -tlnp | grep ':80 ' || sudo ss -tlnp | grep ':80 ' || echo 'Port 80 check failed'"
        self.run_ssh_command(port_check, show_output=True, timeout=100)
        
        print(f"\n{Colors.OKGREEN}[OK] Nginx configured successfully!{Colors.ENDC}\n")
        print(f"{Colors.WARNING}IMPORTANT: Make sure your Excloud security group allows HTTP (port 80) traffic!{Colors.ENDC}")
        print(f"{Colors.WARNING}Security Group ID: {os.getenv('EXCLOUD_SECURITY_GROUP_ID', 'Check your Excloud console')}{Colors.ENDC}\n")
        return True
    
    def setup_env_file(self, record_name: str, ssl_success: bool = True, local_env_path: str = None) -> bool:
        """
        Copy prod.env file to server and set up environment variables.
        
        This copies the prod.env file, updates BASE_URL based on the record_name
        and SSL status, then uploads to the server.
        
        Args:
            record_name: The subdomain record name (e.g., 'api' for api.follow.email)
            ssl_success: Whether SSL was successfully installed (affects protocol)
            local_env_path: Optional path to the local env file
        """
        print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
        print(f"{Colors.HEADER}Setting Up Environment Variables{Colors.ENDC}")
        print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
        
        # Default to prod.env file if not specified
        if local_env_path is None:
            local_env_path = os.path.join(os.path.dirname(__file__), '../prod.env')
        
        # First check if .env exists locally
        if not os.path.exists(local_env_path):
            print(f"{Colors.WARNING}prod.env file not found at {local_env_path}{Colors.ENDC}")
            print(f"{Colors.WARNING}Skipping env setup. You'll need to copy it manually.{Colors.ENDC}")
            return False
        
        # Determine protocol and BASE_URL
        protocol = "https" if ssl_success else "http"
        domain = f"{record_name}.follow.email"
        base_url = f"{protocol}://{domain}"
        
        print(f"{Colors.OKBLUE}Reading prod.env and updating BASE_URL...{Colors.ENDC}")
        print(f"  Protocol: {protocol}")
        print(f"  Domain: {domain}")
        print(f"  BASE_URL: {base_url}")
        
        # Read the env file and update BASE_URL
        try:
            with open(local_env_path, 'r') as f:
                env_content = f.read()
            
            # Update or add BASE_URL
            import re
            if re.search(r'^BASE_URL=.*$', env_content, re.MULTILINE):
                # Replace existing BASE_URL
                env_content = re.sub(
                    r'^BASE_URL=.*$',
                    f'BASE_URL={base_url}',
                    env_content,
                    flags=re.MULTILINE
                )
            else:
                # Add BASE_URL at the end
                env_content += f"\nBASE_URL={base_url}\n"
            
            # Write to a temporary file for upload
            import tempfile
            with tempfile.NamedTemporaryFile(mode='w', suffix='.env', delete=False) as tmp_file:
                tmp_file.write(env_content)
                tmp_env_path = tmp_file.name
            
        except Exception as e:
            print(f"{Colors.FAIL}Failed to process env file: {e}{Colors.ENDC}")
            return False
        
        print(f"{Colors.OKBLUE}Copying updated env file to server...{Colors.ENDC}")
        
        try:
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
                    tmp_env_path,
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
        finally:
            # Clean up temp file
            try:
                os.unlink(tmp_env_path)
            except:
                pass
        
        print(f"\n{Colors.OKGREEN}[OK] Environment variables configured!{Colors.ENDC}")
        print(f"  BASE_URL set to: {base_url}\n")
        return True
    
    def deploy_application(self, github_repo: Optional[str] = None) -> bool:
        """Deploy the application with retry logic"""
        print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
        print(f"{Colors.HEADER}Deploying Application{Colors.ENDC}")
        print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
        
        if not github_repo:
            github_repo = os.getenv('GITHUB_REPO', 'https://github.com/followdotemail/follow.email.git')
        
        commands = [
            CommandConfig(
                description="Creating app directory",
                command="sudo mkdir -p /opt/follow.email && sudo chown $USER:$USER /opt/follow.email",
                criticality=CommandCriticality.CRITICAL,
                timeout=60,
                max_retries=3,
                show_output=True
            ),
            CommandConfig(
                description="Cloning repository",
                command=f"[ -d /opt/follow.email/.git ] && (cd /opt/follow.email && git fetch --all && git reset --hard origin/$(git rev-parse --abbrev-ref HEAD)) || git clone {github_repo} /opt/follow.email",
                criticality=CommandCriticality.CRITICAL,
                timeout=600,
                max_retries=3,
                show_output=True,
                check_running=True
            ),
        ]
        
        success, _ = self.run_commands_batch(commands)
        
        if success:
            print(f"\n{Colors.OKGREEN}[OK] Application deployment base setup complete!{Colors.ENDC}")
        else:
            print(f"\n{Colors.FAIL}[ERROR] Application deployment failed{Colors.ENDC}")
        
        return success
    
    def start_application(self) -> bool:
        """Start the application using docker-compose with retry logic"""
        print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
        print(f"{Colors.HEADER}Starting Application{Colors.ENDC}")
        print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
        
        commands = [
            CommandConfig(
                description="Building Docker image",
                command="cd /opt/follow.email/infra && sudo docker compose build backend",
                criticality=CommandCriticality.CRITICAL,
                timeout=900,  # Docker builds can take a while
                max_retries=2,
                show_output=True,
                check_running=True
            ),
            CommandConfig(
                description="Starting application",
                command="cd /opt/follow.email/infra && sudo docker compose up -d backend",
                criticality=CommandCriticality.CRITICAL,
                timeout=300,
                max_retries=3,
                show_output=True
            ),
            CommandConfig(
                description="Waiting for app to be healthy",
                command="sleep 60",
                criticality=CommandCriticality.OPTIONAL,
                timeout=120,
                show_output=False
            ),
        ]
        
        success, _ = self.run_commands_batch(commands)
        
        if not success:
            print(f"{Colors.FAIL}[ERROR] Failed to start application{Colors.ENDC}")
            return False
        
        # Check container status (optional verification)
        print(f"\n{Colors.OKBLUE}Checking container status...{Colors.ENDC}")
        status_config = CommandConfig(
            description="Container status check",
            command="cd /opt/follow.email/infra && sudo docker compose ps backend",
            criticality=CommandCriticality.OPTIONAL,
            timeout=60,
            show_output=True
        )
        self.run_command_with_retry(status_config)
        
        # Check if the app is responding (optional health check)
        print(f"\n{Colors.OKBLUE}Checking application health...{Colors.ENDC}")
        health_config = CommandConfig(
            description="Application health check",
            command="curl -sf http://localhost:8080/api/v1/health",
            criticality=CommandCriticality.OPTIONAL,
            timeout=30,
            max_retries=3,
            initial_delay=10.0,
            show_output=True
        )
        health_result = self.run_command_with_retry(health_config)
        
        if health_result.success:
            print(f"{Colors.OKGREEN}[OK] Application is running and healthy!{Colors.ENDC}")
        else:
            print(f"{Colors.WARNING}Warning: Health check failed{Colors.ENDC}")
            print(f"{Colors.OKBLUE}Checking container logs...{Colors.ENDC}")
            logs_config = CommandConfig(
                description="Container logs",
                command="cd /opt/follow.email/infra && sudo docker compose logs --tail=50 backend",
                criticality=CommandCriticality.OPTIONAL,
                timeout=120,
                show_output=True
            )
            self.run_command_with_retry(logs_config)
            print(f"{Colors.WARNING}Please check the logs above for errors{Colors.ENDC}")
        
        print(f"\n{Colors.OKGREEN}[OK] Application started successfully!{Colors.ENDC}\n")
        return True

    def get_ssl_cert_dir(self, domain: str) -> str:
        """
        Get the local SSL certificate directory for a domain.
        Directory structure: apps/hermes/deployments/ssl_certs/<domain>/
        """
        base_ssl_dir = os.path.join(os.path.dirname(__file__), 'ssl_certs')
        return os.path.join(base_ssl_dir, domain)

    def download_ssl_certificates(self, record_name: str) -> bool:
        """
        Download SSL certificates from the server to local ssl_certs/<domain>/ directory.
        This preserves certificates for reuse when recreating instances.
        """
        domain = f"{record_name}.follow.email"
        local_cert_dir = self.get_ssl_cert_dir(domain)
        
        print(f"\n{Colors.OKBLUE}Downloading SSL certificates for {domain}...{Colors.ENDC}")
        
        # Create local directory structure
        os.makedirs(local_cert_dir, exist_ok=True)
        
        # Certificate files to download
        cert_files = ['fullchain.pem', 'privkey.pem', 'chain.pem', 'cert.pem']
        remote_cert_dir = f"/etc/letsencrypt/live/{domain}"
        
        downloaded = 0
        for cert_file in cert_files:
            remote_path = f"{remote_cert_dir}/{cert_file}"
            local_path = os.path.join(local_cert_dir, cert_file)
            
            # First, copy to a readable location on server (letsencrypt files need sudo)
            temp_remote = f"/tmp/{cert_file}"
            copy_cmd = f"sudo cp {remote_path} {temp_remote} && sudo chmod 644 {temp_remote}"
            
            if not self.run_ssh_command(copy_cmd, show_output=False, timeout=60):
                if cert_file in ['fullchain.pem', 'privkey.pem']:
                    print(f"{Colors.WARNING}Failed to copy {cert_file} (required){Colors.ENDC}")
                continue
            
            # Download via SCP
            scp_command = [
                'scp',
                '-o', 'StrictHostKeyChecking=no',
                '-o', 'UserKnownHostsFile=/dev/null',
                '-i', self.ssh_key_path,
                f'{self.ssh_user}@{self.ip_address}:{temp_remote}',
                local_path
            ]
            
            try:
                result = subprocess.run(scp_command, capture_output=True, timeout=60)
                if result.returncode == 0:
                    print(f"  {Colors.OKGREEN}[OK] Downloaded {cert_file}{Colors.ENDC}")
                    downloaded += 1
                else:
                    print(f"  {Colors.WARNING}Failed to download {cert_file}{Colors.ENDC}")
            except Exception as e:
                print(f"  {Colors.WARNING}Error downloading {cert_file}: {e}{Colors.ENDC}")
            finally:
                # Clean up temp file on server
                self.run_ssh_command(f"sudo rm -f {temp_remote}", show_output=False, timeout=30)
        
        if downloaded >= 2:  # At least fullchain.pem and privkey.pem
            print(f"\n{Colors.OKGREEN}[OK] SSL certificates saved to {local_cert_dir}{Colors.ENDC}")
            return True
        else:
            print(f"\n{Colors.WARNING}Warning: Could not download all required certificates{Colors.ENDC}")
            return False

    def setup_ssl_certificate(self, record_name: str) -> bool:
        """SSL certificate installation using Let's Encrypt Certbot"""
        print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
        print(f"{Colors.HEADER}Setting up SSL Certificate from - Let's Encrypt Certbot for {record_name}{Colors.ENDC}")
        print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")

        domain = f"{record_name}.follow.email"

        commands = [
            CommandConfig(
                description="Installing certbot",
                command="sudo apt-get install -y certbot python3-certbot-nginx",
                criticality=CommandCriticality.CRITICAL,
                timeout=300,
                max_retries=3,
                show_output=False,
                check_running=True
            ),
            CommandConfig(
                description="Obtaining SSL certificate",
                command=f"sudo certbot --nginx -d {domain} --non-interactive --agree-tos --email admin@follow.email --redirect",
                criticality=CommandCriticality.IMPORTANT,
                timeout=120,
                max_retries=2,
                show_output=True
            ),
        ]

        success, results = self.run_commands_batch(commands)
        
        if not success or (len(results) > 1 and not results[1].success):
            # If certbot fails, it might be because DNS hasn't propagated yet or rate limit
            print(f"{Colors.WARNING}Warning: SSL certificate installation failed{Colors.ENDC}")
            print(f"{Colors.WARNING}This might be because DNS hasn't propagated yet or rate limits.{Colors.ENDC}")
            print(f"{Colors.WARNING}You can run this manually later: sudo certbot --nginx -d {domain}{Colors.ENDC}")
            print(f"{Colors.WARNING}Make sure port 80 is accessible and DNS points to this server.{Colors.ENDC}")
            return False
        
        # Verify certificate was installed successfully
        print(f"\n{Colors.OKBLUE}Verifying SSL certificate...{Colors.ENDC}")

        cert_check = f"sudo certbot certificates | grep -A2 '{domain}' || echo 'Certificate not found!'"
        self.run_ssh_command(cert_check, show_output=True, timeout=300)

        # Test auto renewal (optional)
        print(f"\n{Colors.OKBLUE}Testing certificate auto-renewal...{Colors.ENDC}")
        renewal_config = CommandConfig(
            description="Testing auto-renewal",
            command="sudo certbot renew --dry-run",
            criticality=CommandCriticality.OPTIONAL,
            timeout=100,
            show_output=True
        )
        renewal_result = self.run_command_with_retry(renewal_config)
        if renewal_result.success:
            print(f"{Colors.OKGREEN}[OK] Auto-renewal test passed{Colors.ENDC}")
        else:
            print(f"{Colors.WARNING}Warning: Auto-renewal test failed, but certificate is installed{Colors.ENDC}")

        # Verify nginx config with SSL (important but not critical)
        print(f"\n{Colors.OKBLUE}Verifying nginx SSL configuration...{Colors.ENDC}")
        nginx_config = CommandConfig(
            description="Nginx config test",
            command="sudo nginx -t < /dev/null 2>&1",
            criticality=CommandCriticality.IMPORTANT,
            timeout=100,
            show_output=True
        )
        if self.run_command_with_retry(nginx_config).success:
            print(f"{Colors.OKGREEN}[OK] Nginx SSL configuration is valid{Colors.ENDC}")
        else:
            print(f"{Colors.WARNING}Warning: Nginx configuration test failed{Colors.ENDC}")
        
        # Reload nginx to apply SSL config
        print(f"\n{Colors.OKBLUE}Reloading nginx to apply SSL configuration...{Colors.ENDC}")
        reload_config = CommandConfig(
            description="Nginx reload",
            command="sudo systemctl reload nginx && sleep 2 && sudo systemctl is-active --quiet nginx && echo 'Nginx is active'",
            criticality=CommandCriticality.IMPORTANT,
            timeout=100,
            max_retries=2,
            show_output=True
        )
        if self.run_command_with_retry(reload_config).success:
            print(f"{Colors.OKGREEN}[OK] Nginx reloaded successfully{Colors.ENDC}")
        else:
            print(f"{Colors.WARNING}Warning: Nginx reload failed, but continuing...{Colors.ENDC}")
        
        # Download certificates to local directory for future use
        print(f"\n{Colors.OKBLUE}Saving SSL certificates locally for future use...{Colors.ENDC}")
        self.download_ssl_certificates(record_name)
        
        print(f"\n{Colors.OKGREEN}[OK] SSL certificate installed successfully!{Colors.ENDC}\n")
        print(f"{Colors.WARNING}IMPORTANT: Make sure your Excloud security group allows HTTPS (port 443) traffic!{Colors.ENDC}")
        print(f"{Colors.WARNING}Security Group ID: {os.getenv('EXCLOUD_SECURITY_GROUP_ID', 'Check your Excloud console')}{Colors.ENDC}\n")
        return True


    def install_existing_ssl_certificate(self, record_name: str) -> bool:
        """
        Install existing SSL certificate from local ssl_certs/<domain>/ directory.
        Directory structure: ssl_certs/<domain>/fullchain.pem, privkey.pem, etc.
        """
        domain = f"{record_name}.follow.email"
        local_cert_path = self.get_ssl_cert_dir(domain)

        print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
        print(f"{Colors.HEADER}Installing Existing SSL Certificate for {domain}{Colors.ENDC}")
        print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")

        local_fullchain = os.path.join(local_cert_path, 'fullchain.pem')
        local_privkey = os.path.join(local_cert_path, 'privkey.pem')
        local_chain = os.path.join(local_cert_path, 'chain.pem')
        local_cert = os.path.join(local_cert_path, 'cert.pem')

        required_files = [local_fullchain, local_privkey]

        # Check if certificate directory exists and has required files
        if not os.path.exists(local_cert_path):
            print(f"{Colors.WARNING}Certificate directory not found: {local_cert_path}{Colors.ENDC}")
            return False

        for file_path in required_files:
            if not os.path.exists(file_path):
                print(f"{Colors.FAIL}Required certificate file not found: {file_path}{Colors.ENDC}")
                return False

        print(f"{Colors.OKBLUE}Found existing SSL certificates at {local_cert_path}, uploading to server...{Colors.ENDC}")
        
        # Create letsencrypt directory structure on server using retry logic
        dir_commands = [
            CommandConfig(
                description="Creating certificate directories",
                command=f"sudo mkdir -p /etc/letsencrypt/live/{domain} && sudo mkdir -p /etc/letsencrypt/archive/{domain}",
                criticality=CommandCriticality.CRITICAL,
                timeout=60,
                max_retries=3
            )
        ]
        
        success, _ = self.run_commands_batch(dir_commands)
        if not success:
            print(f"{Colors.FAIL}Failed to create certificate directories{Colors.ENDC}")
            return False

        # Upload certificate files via SCP
        cert_files = [
            (local_fullchain, '/tmp/fullchain.pem', f'/etc/letsencrypt/live/{domain}/fullchain.pem'),
            (local_privkey, '/tmp/privkey.pem', f'/etc/letsencrypt/live/{domain}/privkey.pem'),
        ]

        # Add optional files if they exist
        if os.path.exists(local_chain):
            cert_files.append((local_chain, '/tmp/chain.pem', f'/etc/letsencrypt/live/{domain}/chain.pem'))
        if os.path.exists(local_cert):
            cert_files.append((local_cert, '/tmp/cert.pem', f'/etc/letsencrypt/live/{domain}/cert.pem'))

        for local_file, remote_tmp, remote_final in cert_files:
            # Upload to /tmp first with retry
            max_retries = 3
            for attempt in range(max_retries):
                try:
                    scp_command = [
                        'scp',
                        '-o', 'StrictHostKeyChecking=no',
                        '-o', 'UserKnownHostsFile=/dev/null',
                        '-i', self.ssh_key_path,
                        local_file,
                        f'{self.ssh_user}@{self.ip_address}:{remote_tmp}'
                    ]
                    
                    result = subprocess.run(scp_command, capture_output=True, timeout=100)
                    if result.returncode == 0:
                        break
                    
                    if attempt < max_retries - 1:
                        print(f"{Colors.WARNING}Retry {attempt + 2}/{max_retries} for {os.path.basename(local_file)}...{Colors.ENDC}")
                        time.sleep(5 * (attempt + 1))
                except subprocess.TimeoutExpired:
                    if attempt < max_retries - 1:
                        print(f"{Colors.WARNING}Timeout, retrying {os.path.basename(local_file)}...{Colors.ENDC}")
                        time.sleep(5 * (attempt + 1))
                    continue
            else:
                print(f"{Colors.FAIL}Failed to upload {os.path.basename(local_file)} after {max_retries} attempts{Colors.ENDC}")
                return False
            
            # Move to proper location with correct permissions
            move_config = CommandConfig(
                description=f"Moving {os.path.basename(local_file)}",
                command=f"sudo mv {remote_tmp} {remote_final}",
                criticality=CommandCriticality.CRITICAL,
                timeout=60,
                max_retries=2
            )
            if not self.run_command_with_retry(move_config).success:
                print(f"{Colors.FAIL}Failed to move {os.path.basename(local_file)} to final location{Colors.ENDC}")
                return False
            
            print(f"{Colors.OKGREEN}[OK] Uploaded {os.path.basename(local_file)}{Colors.ENDC}")

        # Set correct permissions (optional - don't fail if these don't work)
        permission_commands = [
            CommandConfig(
                description="Setting certificate permissions",
                command=f"sudo chmod 644 /etc/letsencrypt/live/{domain}/fullchain.pem && "
                        f"sudo chmod 600 /etc/letsencrypt/live/{domain}/privkey.pem && "
                        f"sudo chown -R root:root /etc/letsencrypt/live/{domain}/",
                criticality=CommandCriticality.IMPORTANT,
                timeout=60,
                max_retries=2
            )
        ]
        self.run_commands_batch(permission_commands)
        
        # Update nginx configuration to use SSL
        print(f"\n{Colors.OKBLUE}Updating nginx configuration for SSL...{Colors.ENDC}")
        
        nginx_ssl_config = f"""server {{
    listen 80;
    server_name {domain};
    
    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}}

server {{
    listen 443 ssl http2;
    server_name {domain};

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/{domain}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/{domain}/privkey.pem;
    
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
        
        # Write and upload nginx config with SSL
        import tempfile
        with tempfile.NamedTemporaryFile(mode='w', suffix='.conf', delete=False) as tmp_file:
            tmp_file.write(nginx_ssl_config)
            tmp_config_path = tmp_file.name
        
        try:
            # Upload config with retry
            max_retries = 3
            for attempt in range(max_retries):
                try:
                    scp_command = [
                        'scp',
                        '-o', 'StrictHostKeyChecking=no',
                        '-o', 'UserKnownHostsFile=/dev/null',
                        '-i', self.ssh_key_path,
                        tmp_config_path,
                        f'{self.ssh_user}@{self.ip_address}:/tmp/nginx-ssl-config.tmp'
                    ]
                    result = subprocess.run(scp_command, capture_output=True, timeout=300)
                    if result.returncode == 0:
                        break
                    if attempt < max_retries - 1:
                        time.sleep(5 * (attempt + 1))
                except subprocess.TimeoutExpired:
                    if attempt < max_retries - 1:
                        time.sleep(5 * (attempt + 1))
                    continue
            else:
                print(f"{Colors.FAIL}Failed to upload nginx config after {max_retries} attempts{Colors.ENDC}")
                return False
            
            # Move to nginx sites-available
            move_config = CommandConfig(
                description="Moving nginx SSL config",
                command="sudo mv -f /tmp/nginx-ssl-config.tmp /etc/nginx/sites-available/follow-email && "
                        "sudo chmod 644 /etc/nginx/sites-available/follow-email",
                criticality=CommandCriticality.CRITICAL,
                timeout=60,
                max_retries=2
            )
            self.run_command_with_retry(move_config)
            
        finally:
            try:
                os.unlink(tmp_config_path)
            except:
                pass
        
        # Test and reload nginx
        nginx_commands = [
            CommandConfig(
                description="Testing nginx configuration",
                command="sudo nginx -t < /dev/null 2>&1",
                criticality=CommandCriticality.CRITICAL,
                timeout=60,
                max_retries=2,
                show_output=True
            ),
            CommandConfig(
                description="Reloading nginx",
                command="sudo systemctl reload nginx",
                criticality=CommandCriticality.CRITICAL,
                timeout=60,
                max_retries=2,
                show_output=True
            )
        ]
        
        success, _ = self.run_commands_batch(nginx_commands)
        if not success:
            print(f"{Colors.FAIL}Failed to configure nginx with SSL{Colors.ENDC}")
            return False
        
        print(f"\n{Colors.OKGREEN}[OK] SSL certificate installed successfully from local cache!{Colors.ENDC}\n")
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

    discord = DiscordNotifier()
    
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
            discord.info(title="Follow.Email backend instance provisioning started", description=f"Started provisioning for **{args.record_name}.follow.email**")
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

            # Check for existing SSL certificates in domain-specific directory
            # Directory structure: ssl_certs/<domain>/fullchain.pem, privkey.pem, etc.
            domain = f"{record_name}.follow.email"
            ssl_cert_dir = provisioner.get_ssl_cert_dir(domain)
            ssl_fullchain = os.path.join(ssl_cert_dir, 'fullchain.pem')
            ssl_privkey = os.path.join(ssl_cert_dir, 'privkey.pem')
            
            # Track SSL installation success
            ssl_success = False
            
            if os.path.exists(ssl_fullchain) and os.path.exists(ssl_privkey):
                print(f"{Colors.OKBLUE}Found existing SSL certificates for {domain}, will reuse them...{Colors.ENDC}")
                ssl_success = provisioner.install_existing_ssl_certificate(record_name)
                if not ssl_success:
                    print(f"{Colors.WARNING}Warning: Failed to install existing SSL certificate{Colors.ENDC}")
                    print(f"{Colors.WARNING}Falling back to obtaining new certificate...{Colors.ENDC}")
                    ssl_success = provisioner.setup_ssl_certificate(record_name)
                    if not ssl_success:
                        print(f"{Colors.WARNING}Warning: SSL certificate installation failed or skipped{Colors.ENDC}")
                        print(f"{Colors.WARNING}You can install it manually later when DNS has propagated{Colors.ENDC}")
            else:
                print(f"{Colors.OKBLUE}No existing SSL certificates found for {domain}{Colors.ENDC}")
                print(f"{Colors.OKBLUE}Will obtain new certificate from Let's Encrypt...{Colors.ENDC}")
                # No existing certificates for this domain, obtain new ones
                ssl_success = provisioner.setup_ssl_certificate(record_name)
                if not ssl_success:
                    print(f"{Colors.WARNING}Warning: SSL certificate installation failed or skipped{Colors.ENDC}")
                    print(f"{Colors.WARNING}You can install it manually later when DNS has propagated{Colors.ENDC}")

            # Determine protocol based on SSL success
            protocol = "https" if ssl_success else "http"
            api_url = f"{protocol}://{record_name}.follow.email"

            if not provisioner.deploy_application(args.github_repo):
                print(f"{Colors.FAIL}Failed to deploy application{Colors.ENDC}")
                sys.exit(1)
            
            # Step 5: Setup environment variables with correct BASE_URL
            print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
            print(f"{Colors.HEADER}Step 5: Setting Up Environment{Colors.ENDC}")
            print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
            
            provisioner.setup_env_file(record_name=record_name, ssl_success=ssl_success)
            
            # Step 6: Start the application
            if not provisioner.start_application():
                print(f"{Colors.WARNING}Warning: Failed to start application automatically{Colors.ENDC}")
                print(f"{Colors.WARNING}You can start it manually after provisioning{Colors.ENDC}")
            
            success_msg = f"Follow.Email Backend is now running on **{api_url}**"
            discord.success(
                title="Follow.Email backend instance provisioning complete",
                description=success_msg,
                fields=[
                    {
                        "name": "Application URL",
                        "value":  api_url,
                        "inline": True
                    },
                    {
                        "name": "IP Address",
                        "value":  f"**{instance_ip}**",
                        "inline": True
                    },
                    {
                        "name": "VM ID",
                        "value": f"**{vm_id}**",
                        "inline": True
                    },
                    {
                        "name": "SSL Status",
                        "value": "✅ Enabled" if ssl_success else "❌ Not configured",
                        "inline": True
                    },
                ],
            )
            
            # Success!
            print(f"\n{Colors.HEADER}{'='*60}{Colors.ENDC}")
            print(f"{Colors.OKGREEN}{'[OK] PROVISIONING COMPLETE!':^60}{Colors.ENDC}")
            print(f"{Colors.HEADER}{'='*60}{Colors.ENDC}\n")
            
            print(f"{Colors.BOLD}Instance Details:{Colors.ENDC}")
            print(f"  VM ID:       {vm_id}")
            print(f"  IP Address:  {instance_ip}")
            print(f"  SSH Access:  ssh -i {args.ssh_key} ubuntu@{instance_ip}")
            print(f"  API URL:     {api_url}")
            print(f"  SSL Status:  {'Enabled' if ssl_success else 'Not configured (HTTP only)'}")
            print(f"\n{Colors.BOLD}Quick Commands:{Colors.ENDC}")
            print(f"  Check status: ssh -i {args.ssh_key} ubuntu@{instance_ip} 'cd /opt/follow.email/infra && sudo docker compose ps'")
            print(f"  View logs:    ssh -i {args.ssh_key} ubuntu@{instance_ip} 'cd /opt/follow.email/infra && sudo docker compose logs -f backend'")
            print(f"  Restart app:  ssh -i {args.ssh_key} ubuntu@{instance_ip} 'cd /opt/follow.email/infra && sudo docker compose restart backend'")
            print(f"  Test API:     curl {api_url}/api/v1/health")
            print(f"\n{Colors.OKGREEN}[OK] {success_msg}{Colors.ENDC}\n")
        
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
            if excloud.delete_instance(vm_id):
                discord.success("Instance Destroyed", f"Instance **{vm_id}** has been terminated successfully.")
            else:
                discord.error("Destruction Failed", f"Failed to terminate instance **{vm_id}**.")
        
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
            
            # Check for existing SSL certificates in domain-specific directory
            domain = f"{record_name}.follow.email"
            ssl_cert_dir = provisioner.get_ssl_cert_dir(domain)
            ssl_fullchain = os.path.join(ssl_cert_dir, 'fullchain.pem')
            ssl_privkey = os.path.join(ssl_cert_dir, 'privkey.pem')
            
            # Track SSL success
            ssl_success = False
            
            if os.path.exists(ssl_fullchain) and os.path.exists(ssl_privkey):
                print(f"{Colors.OKBLUE}Found existing SSL certificates for {domain}{Colors.ENDC}")
                ssl_success = provisioner.install_existing_ssl_certificate(record_name)
            else:
                print(f"{Colors.OKBLUE}No existing SSL certificates for {domain}, obtaining new ones...{Colors.ENDC}")
                ssl_success = provisioner.setup_ssl_certificate(record_name)
            
            provisioner.deploy_application(args.github_repo)
            provisioner.setup_env_file(record_name=record_name, ssl_success=ssl_success)
            provisioner.start_application()
            
            # Determine protocol based on SSL success
            protocol = "https" if ssl_success else "http"
            api_url = f"{protocol}://{record_name}.follow.email"
            
            print(f"\n{Colors.OKGREEN}[OK] Setup complete! Application running at {api_url}{Colors.ENDC}")
            print(f"  SSL Status: {'Enabled' if ssl_success else 'Not configured (HTTP only)'}\n")
    
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
