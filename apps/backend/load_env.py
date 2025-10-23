#!/usr/bin/env python3
"""
Load environment variables from .env file
This script reads a .env file and sets all variables as system environment variables.
"""

import os
import sys
import re
from pathlib import Path


def parse_env_line(line):
    """
    Parse a single line from .env file.
    Returns (key, value) tuple or None if line should be skipped.
    """
    # Strip whitespace
    line = line.strip()
    
    # Skip empty lines and comments
    if not line or line.startswith('#'):
        return None
    
    # Match KEY=VALUE pattern
    match = re.match(r'^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$', line)
    if not match:
        return None
    
    key = match.group(1)
    value = match.group(2)
    
    # Remove quotes if present (both single and double quotes)
    if value:
        # Remove surrounding quotes
        if (value.startswith('"') and value.endswith('"')) or \
           (value.startswith("'") and value.endswith("'")):
            value = value[1:-1]
    
    return (key, value)


def load_env_file(filepath='.env'):
    """
    Load environment variables from .env file.
    Returns the number of variables loaded.
    """
    # Convert to absolute path if relative
    if not os.path.isabs(filepath):
        # Get the directory where this script is located
        script_dir = Path(__file__).parent.resolve()
        filepath = script_dir / filepath
    
    if not os.path.exists(filepath):
        print(f"Error: {filepath} file not found!")
        return 0
    
    loaded_count = 0
    
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            for line_num, line in enumerate(f, 1):
                result = parse_env_line(line)
                
                if result:
                    key, value = result
                    # Set environment variable
                    os.environ[key] = value
                    loaded_count += 1
                    print(f"[OK] Loaded: {key}")
        
        print(f"\n[SUCCESS] Successfully loaded {loaded_count} environment variables from {filepath}")
        return loaded_count
    
    except Exception as e:
        print(f"[ERROR] Error reading {filepath}: {e}")
        return 0


def display_loaded_variables(env_file='.env'):
    """Display all environment variables from .env file."""
    print("\n" + "="*60)
    print("LOADED ENVIRONMENT VARIABLES:")
    print("="*60)
    
    # Convert to absolute path if relative
    if not os.path.isabs(env_file):
        script_dir = Path(__file__).parent.resolve()
        env_file = script_dir / env_file
    
    # Read .env file to get the keys we loaded
    if os.path.exists(env_file):
        with open(env_file, 'r', encoding='utf-8') as f:
            for line in f:
                result = parse_env_line(line)
                if result:
                    key, _ = result
                    value = os.environ.get(key, '')
                    # Mask sensitive values (show only first 4 chars)
                    if any(sensitive in key.upper() for sensitive in ['SECRET', 'KEY', 'PASSWORD', 'TOKEN']):
                        masked_value = value[:4] + '*' * (len(value) - 4) if len(value) > 4 else '****'
                        print(f"{key:30} = {masked_value}")
                    else:
                        print(f"{key:30} = {value}")
    print("="*60)


def generate_batch_file(env_file='.env', output_file='set_env.bat'):
    """Generate a Windows batch file to set environment variables."""
    # Convert to absolute path if relative
    env_file_path = env_file
    if not os.path.isabs(env_file):
        script_dir = Path(__file__).parent.resolve()
        env_file_path = script_dir / env_file
        output_file = script_dir / output_file
    
    if not os.path.exists(env_file_path):
        print(f"[ERROR] {env_file_path} not found!")
        return False
    
    try:
        with open(output_file, 'w', encoding='utf-8') as bat_file:
            bat_file.write('@echo off\n')
            bat_file.write('REM Auto-generated batch file to set environment variables\n')
            bat_file.write('REM Generated from .env file\n\n')
            
            with open(env_file_path, 'r', encoding='utf-8') as env:
                for line in env:
                    result = parse_env_line(line)
                    if result:
                        key, value = result
                        # Escape special characters for batch
                        escaped_value = value.replace('%', '%%').replace('^', '^^')
                        bat_file.write(f'set {key}={escaped_value}\n')
            
            bat_file.write('\necho Environment variables loaded successfully!\n')
        
        print(f"\n[SUCCESS] Generated batch file: {output_file}")
        print(f"\nTo set environment variables in your current shell, run:")
        print(f"    {output_file}")
        return True
    
    except Exception as e:
        print(f"[ERROR] Failed to generate batch file: {e}")
        return False


def main():
    """Main function to load environment variables."""
    print("="*60)
    print("Environment Variable Loader")
    print("="*60)
    
    # Parse arguments
    env_file = '.env'
    for arg in sys.argv[1:]:
        if not arg.startswith('-'):
            env_file = arg
            break
    
    # Load environment variables (only in Python process)
    count = load_env_file(env_file)
    
    if count > 0:
        # Option to display loaded variables
        if '--show' in sys.argv or '-s' in sys.argv:
            display_loaded_variables(env_file)
        
        # Option to export for shell
        if '--export' in sys.argv or '-e' in sys.argv:
            print("\n" + "="*60)
            print("EXPORT COMMANDS FOR BASH/SH:")
            print("="*60)
            
            # Convert to absolute path if relative
            env_file_path = env_file
            if not os.path.isabs(env_file):
                script_dir = Path(__file__).parent.resolve()
                env_file_path = script_dir / env_file
            
            if os.path.exists(env_file_path):
                with open(env_file_path, 'r', encoding='utf-8') as f:
                    for line in f:
                        result = parse_env_line(line)
                        if result:
                            key, value = result
                            # Escape value for shell
                            escaped_value = value.replace('"', '\\"')
                            print(f'export {key}="{escaped_value}"')
            print("="*60)
        
        # Generate Windows batch file
        if '--batch' in sys.argv or '-b' in sys.argv or os.name == 'nt':
            generate_batch_file(env_file)
        
        print("\n" + "="*60)
        print("NOTE: Environment variables set by this script are only")
        print("available within the Python process. To set them in your")
        print("shell, use one of these methods:")
        print("="*60)
        if os.name == 'nt':  # Windows
            print("\nWindows CMD:")
            print("    python load_env.py --batch")
            print("    set_env.bat")
            print("\nWindows PowerShell:")
            print("    Get-Content .env | ForEach-Object {")
            print("        if ($_ -match '^([^#].+?)=(.*)$') {")
            print("            [Environment]::SetEnvironmentVariable($matches[1], $matches[2], 'Process')")
            print("        }")
            print("    }")
        else:  # Unix-like
            print("\nBash/Zsh:")
            print("    export $(cat .env | grep -v '^#' | xargs)")
            print("    # or")
            print("    set -a && source .env && set +a")
        print("="*60)
    
    return 0 if count > 0 else 1


if __name__ == '__main__':
    sys.exit(main())

