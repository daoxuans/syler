#!/bin/bash

# Create required directories if they don't exist
mkdir -p /opt/syler/www/portal/assets/{css,js}

# Set ownership
chown -R www-data:www-data /opt/syler/www

# Set permissions
chmod -R 755 /opt/syler/www

# Create symlink for nginx config
ln -sf /opt/syler/www/portal.conf /etc/nginx/conf.d/

# Test nginx configuration
nginx -t

# Restart nginx if test is successful
if [ $? -eq 0 ]; then
    systemctl restart nginx
    echo "Setup completed successfully"
else
    echo "Nginx configuration test failed"
    exit 1
fi