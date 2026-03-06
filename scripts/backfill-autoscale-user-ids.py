#!/usr/bin/env python3
"""
Backfill user_id field in spawn-autoscale-groups-production table.

For each autoscale group, finds instances with the spawn:autoscale-group tag
and extracts the spawn:iam-user tag to populate the user_id field.
"""

import sys
import boto3
from botocore.exceptions import ClientError

def main():
    if len(sys.argv) != 3:
        print("Usage: backfill-autoscale-user-ids.py <dynamodb-profile> <ec2-profile>")
        print("Example: backfill-autoscale-user-ids.py mycelium-infra mycelium-dev")
        sys.exit(1)

    dynamodb_profile = sys.argv[1]
    ec2_profile = sys.argv[2]
    region = 'us-east-1'

    # Initialize clients with separate profiles
    dynamodb_session = boto3.Session(profile_name=dynamodb_profile, region_name=region)
    ec2_session = boto3.Session(profile_name=ec2_profile, region_name=region)
    dynamodb = dynamodb_session.client('dynamodb')
    ec2 = ec2_session.client('ec2')

    print(f"Scanning spawn-autoscale-groups-production table...")

    # Get all autoscale groups
    try:
        response = dynamodb.scan(TableName='spawn-autoscale-groups-production')
        groups = response['Items']
    except ClientError as e:
        print(f"Error scanning table: {e}")
        sys.exit(1)

    print(f"Found {len(groups)} autoscale groups")

    updated = 0
    skipped = 0
    errors = 0

    for group in groups:
        group_id = group['autoscale_group_id']['S']

        # Check if user_id already exists
        if 'user_id' in group and group['user_id'].get('S'):
            print(f"  Skip: {group_id} (user_id already set)")
            skipped += 1
            continue

        # Find instances with this autoscale group tag
        try:
            instances_response = ec2.describe_instances(
                Filters=[
                    {'Name': 'tag:spawn:autoscale-group', 'Values': [group_id]},
                    {'Name': 'instance-state-name', 'Values': ['running', 'stopped', 'pending']}
                ]
            )
        except ClientError as e:
            print(f"  Error: {group_id} - Failed to query instances: {e}")
            errors += 1
            continue

        # Extract user_id from first instance
        user_id = None
        if instances_response['Reservations']:
            for reservation in instances_response['Reservations']:
                for instance in reservation['Instances']:
                    tags = instance.get('Tags', [])
                    for tag in tags:
                        if tag['Key'] == 'spawn:iam-user':
                            user_id = tag['Value']
                            break
                    if user_id:
                        break
                if user_id:
                    break

        if not user_id:
            print(f"  Warn: {group_id} - No instances with spawn:iam-user tag found")
            errors += 1
            continue

        # Update group with user_id
        try:
            dynamodb.update_item(
                TableName='spawn-autoscale-groups-production',
                Key={'autoscale_group_id': {'S': group_id}},
                UpdateExpression='SET user_id = :uid',
                ExpressionAttributeValues={':uid': {'S': user_id}}
            )
            print(f"  ✓ Updated {group_id} with user {user_id}")
            updated += 1
        except ClientError as e:
            print(f"  Error: {group_id} - Failed to update: {e}")
            errors += 1

    print("")
    print("Summary:")
    print(f"  Updated: {updated}")
    print(f"  Skipped (already set): {skipped}")
    print(f"  Errors: {errors}")

    if errors > 0:
        sys.exit(1)

if __name__ == '__main__':
    main()
