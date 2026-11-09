const database = process.env.MONGO_DATABASE;

db.getSiblingDB(database).createUser({
  user: process.env.MONGO_USER,
  pwd: process.env.MONGO_PASSWORD,
  roles: [{ role: 'readWrite', db: database }],
});
