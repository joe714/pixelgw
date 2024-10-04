import createClient from "openapi-fetch";
import type { paths } from "@/openapi";

export const restClient = createClient<paths>({ baseUrl: "/api" });
