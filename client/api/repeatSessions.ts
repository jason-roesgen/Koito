import type { ImageList, PaginatedResponse } from "./api";

export type RepeatSession = {
  track: {
    id: number;
    title: string;
    artists: { id: number; name: string }[];
    image: ImageList;
  };
  started_at: string;
  ended_at: string;
  play_count: number;
  estimated_seconds: number | null;
  elapsed_seconds: number;
};

export type RepeatSessionPage = PaginatedResponse<RepeatSession> & {
  minimum_plays: number;
  overview: {
    session_count: number;
    estimated_seconds: number;
    sessions_with_estimates: number;
    longest_session: RepeatSession | null;
  };
};
