output "hub_public_ip" {
  description = "Public IP of the mesh hub (WireGuard endpoint + rendezvous)"
  value       = var.allocate_eip ? aws_eip.hub[0].public_ip : aws_instance.hub.public_ip
}

output "hub_private_ip" {
  description = "Private IP of the hub instance"
  value       = aws_instance.hub.private_ip
}

output "hub_wg_interface" {
  description = "WireGuard interface name configured on the hub"
  value       = "wg0"
}

output "hub_security_group" {
  description = "Security group ID (51820/udp + 443/tcp)"
  value       = aws_security_group.hub.id
}
