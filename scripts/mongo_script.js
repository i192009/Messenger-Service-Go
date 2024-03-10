const mongo = require('mongodb');
const MongoClient = mongo.MongoClient;
const url ="mongodb://127.0.0.1:27017";
MongoClient.connect(url).then( async client => {
    const db = client.db('MessageCenter');

    // Create message template collection with Indexes
    const message_template = db.collection('message_templates');
    let result = await message_template.createIndex({name: "text"});
    console.log(`Index created: ${result}`);

    // Create message template collection with Indexes
    const message_class_configuration = db.collection('message_class_configurations');
    result = await message_class_configuration.createIndex({name: "text"});
    console.log(`Index created: ${result}`);
    result = await message_class_configuration.createIndex({appid: 1});
    console.log(`Index created: ${result}`);

    client.close().then(() => {
        console.log('close');
    });
});