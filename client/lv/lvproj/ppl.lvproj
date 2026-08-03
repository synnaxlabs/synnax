<?xml version='1.0' encoding='UTF-8' ?>
<Project Type="Project" LVVersion="26008000">
	<Property Name="NI.LV.All.SaveVersion" Type="Str">26.0</Property>
	<Property Name="NI.LV.All.SourceOnly" Type="Bool">true</Property>
	<Item Name="My Computer" Type="My Computer">
		<Property Name="server.app.propertiesEnabled" Type="Bool">true</Property>
		<Property Name="server.control.propertiesEnabled" Type="Bool">true</Property>
		<Property Name="server.tcp.enabled" Type="Bool">false</Property>
		<Property Name="server.tcp.port" Type="Int">0</Property>
		<Property Name="server.tcp.serviceName" Type="Str">My Computer/VI Server</Property>
		<Property
            Name="server.tcp.serviceName.default"
            Type="Str"
        >My Computer/VI Server</Property>
		<Property Name="server.vi.callsEnabled" Type="Bool">true</Property>
		<Property Name="server.vi.propertiesEnabled" Type="Bool">true</Property>
		<Property Name="specify.custom.address" Type="Bool">false</Property>
		<Item Name="synnax_lib.lvlib" Type="Library" URL="../lvlib/synnax_lib.lvlib" />
		<Item Name="Dependencies" Type="Dependencies" />
		<Item Name="Build Specifications" Type="Build">
			<Item Name="synnax_lib.lvlibp" Type="Packed Library">
				<Property
                    Name="Bld_buildCacheID"
                    Type="Str"
                >{36C33AB3-40B3-48B2-865C-CF9D06E8BAD1}</Property>
				<Property Name="Bld_buildSpecName" Type="Str">synnax_lib.lvlibp</Property>
				<Property Name="Bld_excludeLibraryItems" Type="Bool">true</Property>
				<Property Name="Bld_excludePolymorphicVIs" Type="Bool">true</Property>
				<Property Name="Bld_localDestDir" Type="Path">../builds/synnax_lib.lvlibp</Property>
				<Property Name="Bld_localDestDirType" Type="Str">relativeToProject</Property>
				<Property Name="Bld_modifyLibraryFile" Type="Bool">true</Property>
				<Property
                    Name="Bld_previewCacheID"
                    Type="Str"
                >{8C9EB914-64D5-4529-AAEE-674315A2D267}</Property>
				<Property Name="Bld_version.major" Type="Int">1</Property>
				<Property Name="Destination[0].destName" Type="Str">synnax_lib.lvlibp</Property>
				<Property
                    Name="Destination[0].path"
                    Type="Path"
                >../builds/synnax_lib.lvlibp/synnax_lib.lvlibp</Property>
				<Property Name="Destination[0].path.type" Type="Str">relativeToProject</Property>
				<Property Name="Destination[0].preserveHierarchy" Type="Bool">true</Property>
				<Property Name="Destination[0].type" Type="Str">App</Property>
				<Property Name="Destination[1].destName" Type="Str">Support Directory</Property>
				<Property Name="Destination[1].path" Type="Path">../builds/synnax_lib.lvlibp</Property>
				<Property Name="Destination[1].path.type" Type="Str">relativeToProject</Property>
				<Property Name="DestinationCount" Type="Int">2</Property>
				<Property Name="PackedLib_callersAdapt" Type="Bool">true</Property>
				<Property
                    Name="Source[0].itemID"
                    Type="Str"
                >{22053070-ED5A-4B1C-9216-E732BEA40D36}</Property>
				<Property Name="Source[0].type" Type="Str">Container</Property>
				<Property Name="Source[1].destinationIndex" Type="Int">0</Property>
				<Property Name="Source[1].itemID" Type="Ref">/My Computer/synnax_lib.lvlib</Property>
				<Property Name="Source[1].Library.allowMissingMembers" Type="Bool">true</Property>
				<Property Name="Source[1].Library.atomicCopy" Type="Bool">true</Property>
				<Property Name="Source[1].Library.LVLIBPtopLevel" Type="Bool">true</Property>
				<Property Name="Source[1].preventRename" Type="Bool">true</Property>
				<Property Name="Source[1].sourceInclusion" Type="Str">TopLevel</Property>
				<Property Name="Source[1].type" Type="Str">Library</Property>
				<Property
                    Name="Source[10].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Client.lvclass/Private/synnax channel retrieve keys - Single.vi</Property>
				<Property Name="Source[10].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[10].properties[0].value" Type="Bool">false</Property>
				<Property
                    Name="Source[10].properties[1].type"
                    Type="Str"
                >Remove block diagram</Property>
				<Property Name="Source[10].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[10].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[10].type" Type="Str">VI</Property>
				<Property
                    Name="Source[11].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Client.lvclass/Private/synnax channel retrieve keys.vi</Property>
				<Property Name="Source[11].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[11].properties[0].value" Type="Bool">false</Property>
				<Property
                    Name="Source[11].properties[1].type"
                    Type="Str"
                >Remove block diagram</Property>
				<Property Name="Source[11].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[11].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[11].type" Type="Str">VI</Property>
				<Property
                    Name="Source[12].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Client.lvclass/Private/synnax client close.vi</Property>
				<Property Name="Source[12].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[12].properties[0].value" Type="Bool">false</Property>
				<Property
                    Name="Source[12].properties[1].type"
                    Type="Str"
                >Remove block diagram</Property>
				<Property Name="Source[12].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[12].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[12].type" Type="Str">VI</Property>
				<Property
                    Name="Source[13].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Client.lvclass/Private/synnax client open.vi</Property>
				<Property Name="Source[13].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[13].properties[0].value" Type="Bool">false</Property>
				<Property
                    Name="Source[13].properties[1].type"
                    Type="Str"
                >Remove block diagram</Property>
				<Property Name="Source[13].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[13].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[13].type" Type="Str">VI</Property>
				<Property
                    Name="Source[14].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Client.lvclass/Public/Close.vi</Property>
				<Property Name="Source[14].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[14].properties[0].value" Type="Bool">false</Property>
				<Property
                    Name="Source[14].properties[1].type"
                    Type="Str"
                >Remove block diagram</Property>
				<Property Name="Source[14].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[14].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[14].type" Type="Str">VI</Property>
				<Property
                    Name="Source[15].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Client.lvclass/Public/Open.vi</Property>
				<Property Name="Source[15].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[15].properties[0].value" Type="Bool">false</Property>
				<Property
                    Name="Source[15].properties[1].type"
                    Type="Str"
                >Remove block diagram</Property>
				<Property Name="Source[15].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[15].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[15].type" Type="Str">VI</Property>
				<Property
                    Name="Source[16].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Client.lvclass/Public/Retrieve.vi</Property>
				<Property Name="Source[16].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[16].properties[0].value" Type="Bool">false</Property>
				<Property
                    Name="Source[16].properties[1].type"
                    Type="Str"
                >Remove block diagram</Property>
				<Property Name="Source[16].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[16].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[16].type" Type="Str">VI</Property>
				<Property
                    Name="Source[17].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Client.lvclass/Public/Version.vi</Property>
				<Property Name="Source[17].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[17].properties[0].value" Type="Bool">false</Property>
				<Property
                    Name="Source[17].properties[1].type"
                    Type="Str"
                >Remove block diagram</Property>
				<Property Name="Source[17].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[17].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[17].type" Type="Str">VI</Property>
				<Property
                    Name="Source[18].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Writer.lvclass/Accessors/Read handle.vi</Property>
				<Property Name="Source[18].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[18].properties[0].value" Type="Bool">false</Property>
				<Property
                    Name="Source[18].properties[1].type"
                    Type="Str"
                >Remove block diagram</Property>
				<Property Name="Source[18].properties[1].value" Type="Bool">true</Property>
				<Property Name="Source[18].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[18].type" Type="Str">VI</Property>
				<Property
                    Name="Source[19].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Writer.lvclass/Accessors/Write handle.vi</Property>
				<Property Name="Source[19].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[19].properties[0].value" Type="Bool">false</Property>
				<Property
                    Name="Source[19].properties[1].type"
                    Type="Str"
                >Remove block diagram</Property>
				<Property Name="Source[19].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[19].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[19].type" Type="Str">VI</Property>
				<Property Name="Source[2].Container.applySaveSettings" Type="Bool">true</Property>
				<Property Name="Source[2].Container.depDestIndex" Type="Int">0</Property>
				<Property
                    Name="Source[2].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes</Property>
				<Property Name="Source[2].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[2].properties[0].value" Type="Bool">false</Property>
				<Property Name="Source[2].properties[1].type" Type="Str">Remove block diagram</Property>
				<Property Name="Source[2].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[2].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[2].type" Type="Str">Container</Property>
				<Property Name="Source[20].Container.applySaveSettings" Type="Bool">true</Property>
				<Property Name="Source[20].Container.depDestIndex" Type="Int">0</Property>
				<Property
                    Name="Source[20].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/Examples</Property>
				<Property Name="Source[20].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[20].properties[0].value" Type="Bool">false</Property>
				<Property
                    Name="Source[20].properties[1].type"
                    Type="Str"
                >Remove block diagram</Property>
				<Property Name="Source[20].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[20].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[20].type" Type="Str">Container</Property>
				<Property Name="Source[21].Container.applyProperties" Type="Bool">true</Property>
				<Property Name="Source[21].Container.depDestIndex" Type="Int">0</Property>
				<Property
                    Name="Source[21].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Writer.lvclass/Private/Writes</Property>
				<Property Name="Source[21].properties[0].type" Type="Str">Allow debugging</Property>
				<Property Name="Source[21].properties[0].value" Type="Bool">true</Property>
				<Property Name="Source[21].propertiesCount" Type="Int">1</Property>
				<Property Name="Source[21].type" Type="Str">Container</Property>
				<Property Name="Source[3].Container.applySaveSettings" Type="Bool">true</Property>
				<Property Name="Source[3].Container.depDestIndex" Type="Int">0</Property>
				<Property
                    Name="Source[3].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/private</Property>
				<Property Name="Source[3].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[3].properties[0].value" Type="Bool">false</Property>
				<Property Name="Source[3].properties[1].type" Type="Str">Remove block diagram</Property>
				<Property Name="Source[3].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[3].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[3].type" Type="Str">Container</Property>
				<Property Name="Source[4].Container.applySaveSettings" Type="Bool">true</Property>
				<Property Name="Source[4].Container.depDestIndex" Type="Int">0</Property>
				<Property
                    Name="Source[4].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/public</Property>
				<Property Name="Source[4].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[4].properties[0].value" Type="Bool">false</Property>
				<Property Name="Source[4].properties[1].type" Type="Str">Remove block diagram</Property>
				<Property Name="Source[4].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[4].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[4].type" Type="Str">Container</Property>
				<Property Name="Source[5].Container.applySaveSettings" Type="Bool">true</Property>
				<Property Name="Source[5].Container.depDestIndex" Type="Int">0</Property>
				<Property
                    Name="Source[5].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/typedefs</Property>
				<Property Name="Source[5].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[5].properties[0].value" Type="Bool">false</Property>
				<Property Name="Source[5].properties[1].type" Type="Str">Remove block diagram</Property>
				<Property Name="Source[5].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[5].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[5].type" Type="Str">Container</Property>
				<Property Name="Source[6].Container.applySaveSettings" Type="Bool">true</Property>
				<Property Name="Source[6].Container.depDestIndex" Type="Int">0</Property>
				<Property
                    Name="Source[6].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Writer.lvclass/Private</Property>
				<Property Name="Source[6].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[6].properties[0].value" Type="Bool">false</Property>
				<Property Name="Source[6].properties[1].type" Type="Str">Remove block diagram</Property>
				<Property Name="Source[6].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[6].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[6].type" Type="Str">Container</Property>
				<Property
                    Name="Source[7].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Client.lvclass/Accessors/Read handle.vi</Property>
				<Property Name="Source[7].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[7].properties[0].value" Type="Bool">false</Property>
				<Property Name="Source[7].properties[1].type" Type="Str">Remove block diagram</Property>
				<Property Name="Source[7].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[7].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[7].type" Type="Str">VI</Property>
				<Property
                    Name="Source[8].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Client.lvclass/Accessors/Write handle.vi</Property>
				<Property Name="Source[8].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[8].properties[0].value" Type="Bool">false</Property>
				<Property Name="Source[8].properties[1].type" Type="Str">Remove block diagram</Property>
				<Property Name="Source[8].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[8].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[8].type" Type="Str">VI</Property>
				<Property
                    Name="Source[9].itemID"
                    Type="Ref"
                >/My Computer/synnax_lib.lvlib/classes/Synnax Client.lvclass/Private/synnax channel retrieve keys - Batch.vi</Property>
				<Property Name="Source[9].properties[0].type" Type="Str">Remove front panel</Property>
				<Property Name="Source[9].properties[0].value" Type="Bool">false</Property>
				<Property Name="Source[9].properties[1].type" Type="Str">Remove block diagram</Property>
				<Property Name="Source[9].properties[1].value" Type="Bool">false</Property>
				<Property Name="Source[9].propertiesCount" Type="Int">2</Property>
				<Property Name="Source[9].type" Type="Str">VI</Property>
				<Property Name="SourceCount" Type="Int">22</Property>
				<Property Name="TgtF_companyName" Type="Str">Synnax Labs Inc.</Property>
				<Property Name="TgtF_fileDescription" Type="Str">synnax_lib.lvlibp</Property>
				<Property Name="TgtF_internalName" Type="Str">synnax_lib.lvlibp</Property>
				<Property
                    Name="TgtF_legalCopyright"
                    Type="Str"
                >Copyright © 2026 Synnax Labs Inc.</Property>
				<Property Name="TgtF_productName" Type="Str">synnax_lib.lvlibp</Property>
				<Property
                    Name="TgtF_targetfileGUID"
                    Type="Str"
                >{15676A33-51C1-42E6-9430-AB16465A3F92}</Property>
				<Property Name="TgtF_targetfileName" Type="Str">synnax_lib.lvlibp</Property>
				<Property Name="TgtF_versionIndependent" Type="Bool">true</Property>
			</Item>
		</Item>
	</Item>
</Project>
