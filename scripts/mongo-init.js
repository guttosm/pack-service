// MongoDB initialization script
// This script runs when the container starts for the first time

// Create the application database and user
db = db.getSiblingDB(process.env.MONGO_INITDB_DATABASE || 'pack_service');

// Create application user with read/write permissions
db.createUser({
  user: process.env.MONGO_APP_USER || 'pack_user',
  pwd: process.env.MONGO_APP_PASSWORD || 'pack_password',
  roles: [
    {
      role: 'readWrite',
      db: process.env.MONGO_INITDB_DATABASE || 'pack_service'
    }
  ]
});

print('MongoDB initialization completed successfully');
print('Database: ' + (process.env.MONGO_INITDB_DATABASE || 'pack_service'));
print('User: ' + (process.env.MONGO_APP_USER || 'pack_user'));
