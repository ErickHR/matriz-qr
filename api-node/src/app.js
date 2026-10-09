import express from 'express';
import statisticsRouter from './routes/statistics.routes.js';

export function createApp() {
  const app = express();
  app.use(express.json());

  app.get('/health', (_request, response) => response.status(200).json({ status: 'ok' }));
  app.use('/api/statistics', statisticsRouter);

  return app;
}
