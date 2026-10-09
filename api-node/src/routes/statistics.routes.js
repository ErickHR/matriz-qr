import { Router } from 'express';
import { getStatistics } from '../controllers/statistics.controller.js';

const router = Router();

router.post('/', getStatistics);

export default router;
