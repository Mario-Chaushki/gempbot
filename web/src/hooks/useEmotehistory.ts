import { useEffect, useRef, useState } from "react";
import { doFetch, Method } from "../service/doFetch";
import { useStore } from "../store";

export enum ChangeType {
    ADD = "add",
    REMOVE = "remove",
    REMOVED_RANDOM = "removed_random",
}

export enum EmoteType {
    BTTV = "bttv",
    SEVENTV = "seventv",
}

export interface RawEmotehistoryItem {
    CreatedAt: string;
    UpdatedAt: string;
    DeletedAt: string | null;
    ID: number;
    ChannelTwitchID: string;
    Type: EmoteType;
    ChangeType: ChangeType;
    EmoteID: string;
    Blocked: boolean;
}

export interface EmotehistoryItem {
    CreatedAt: Date;
    UpdatedAt: Date;
    DeletedAt: Date | null;
    ID: number;
    ChannelTwitchID: string;
    Type: EmoteType;
    ChangeType: ChangeType;
    EmoteID: string;
    Blocked: boolean;
}


const PAGE_SIZE = 20;

export function useEmotehistory(added: boolean, channel?: string): [Array<EmotehistoryItem>, () => void, boolean, number, () => void, () => void, (emoteId: string) => void, (emoteId: string) => void] {
    const [page, setPage] = useState(1);
    const pageRef = useRef(page);
    pageRef.current = page;

    const [emotehistory, setEmotehistory] = useState<Array<EmotehistoryItem>>([]);
    const [loading, setLoading] = useState(false);
    const managing = useStore(state => state.managing);
    const apiBaseUrl = useStore(state => state.apiBaseUrl);
    const scToken = useStore(state => state.scToken);

    const fetchPredictions = () => {
        setLoading(true);
        const currentPage = pageRef.current;

        const endPoint = "/api/emotehistory";
        const searchParams = new URLSearchParams();
        if (channel) {
            searchParams.append("channel", channel);
        }
        searchParams.append("page", page.toString());
        if (added) {
            searchParams.append("added", "1");
        }
        doFetch({apiBaseUrl, managing, scToken }, Method.GET, endPoint, searchParams).then((resp) => {
            if (currentPage !== pageRef.current) {
                throw new Error("Page changed");
            }

            return resp
        }).then((items: Array<RawEmotehistoryItem>) =>
            setEmotehistory(
                items.map(
                    (item: RawEmotehistoryItem) => (
                        {
                            ...item,
                            CreatedAt: new Date(item.CreatedAt),
                            UpdatedAt: new Date(item.CreatedAt),
                            DeletedAt: item.CreatedAt ? new Date(item.CreatedAt) : null,
                        }
                    )
                )
            )
        ).then(() => setLoading(false)).catch(err => {
            if (err.message !== "Page changed") {
                throw err;
            }
        });
    };

    const remove = (emoteId: string) => {
        setLoading(true);
        const endPoint = "/api/emotehistory";
        const searchParams = new URLSearchParams();
        searchParams.append("emoteId", emoteId);
        doFetch({apiBaseUrl, managing, scToken }, Method.DELETE, endPoint, searchParams).then(() => {
            fetchPredictions();
        });
    };

    const block = (emoteId: string) => {
        setLoading(true);
        const endPoint = "/api/emotehistory";
        const searchParams = new URLSearchParams();
        searchParams.append("emoteId", emoteId);
        doFetch({apiBaseUrl, managing, scToken }, Method.PATCH, endPoint, searchParams).then(() => {
            fetchPredictions();
        });
    };

    // eslint-disable-next-line react-hooks/exhaustive-deps
    useEffect(fetchPredictions, [managing, page]);

    return [emotehistory, fetchPredictions, loading, page, () => emotehistory.length === PAGE_SIZE ? setPage(page + 1) : undefined, () => page > 1 ? setPage(page - 1) : undefined, remove, block];
}