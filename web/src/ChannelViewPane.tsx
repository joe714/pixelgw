import { useDefaultServiceGetChannels } from "../openapi/queries";

function ChannelViewPane() {
    const { data } = useDefaultServiceGetChannels();

    return (
	<ul>
	  {data?.map((e) => <li key={e.uuid}>{e.name} - {e.comment}</li>)}
        </ul>
    );
}

export default ChannelViewPane;
